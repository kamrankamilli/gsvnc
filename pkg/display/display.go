package display

import (
	"image"

	"github.com/kamrankamilli/gsvnc/pkg/buffer"
	"github.com/kamrankamilli/gsvnc/pkg/display/providers"
	"github.com/kamrankamilli/gsvnc/pkg/rfb/encodings"
	"github.com/kamrankamilli/gsvnc/pkg/rfb/types"
)

// Display manages the local display session.
type Display struct {
	displayProvider providers.Display

	width, height    int
	pixelFormat      *types.PixelFormat
	getEncodingsFunc GetEncodingsFunc
	encodings        []int32
	pseudoEncodings  []int32
	currentEnc       encodings.Encoding

	// Read/writer for the connected client
	buf *buffer.ReadWriter

	// Incoming event queues
	fbReqQueue chan *types.FrameBufferUpdateRequest
	ptrEvQueue chan *types.PointerEvent
	keyEvQueue chan *types.KeyEvent
	cutTxtEvsQ chan *types.ClientCutText

	// Memory of keys that are currently down.
	downKeys []uint32
	// scratch output buffer reused for frames
	outBuf []byte

	lastBtnMask uint8

	// closed to stop watcher goroutines
	done chan struct{}
}

// DefaultPixelFormat used in ServerInit messages (16bpp 5-6-5 true colour).
var DefaultPixelFormat = &types.PixelFormat{
	BPP:        16,
	Depth:      16,
	BigEndian:  0,
	TrueColour: 1,
	RedMax:     0x1f,
	GreenMax:   0x3f,
	BlueMax:    0x1f,
	RedShift:   11,
	GreenShift: 5,
	BlueShift:  0,
}

// GetEncodingsFunc retrieves an encoder from client options.
type GetEncodingsFunc func(encs []int32) encodings.Encoding

// Opts represents options for building a new display.
type Opts struct {
	DisplayProvider providers.Provider
	Width, Height   int
	Buffer          *buffer.ReadWriter
	GetEncodingFunc GetEncodingsFunc
}

// NewDisplay returns a new display with the given dimensions.
func NewDisplay(opts *Opts) *Display {
	return &Display{
		displayProvider:  providers.GetDisplayProvider(opts.DisplayProvider),
		width:            opts.Width,
		height:           opts.Height,
		buf:              opts.Buffer,
		getEncodingsFunc: opts.GetEncodingFunc,
		pixelFormat:      DefaultPixelFormat,
		fbReqQueue:       make(chan *types.FrameBufferUpdateRequest, 128),
		ptrEvQueue:       make(chan *types.PointerEvent, 128),
		keyEvQueue:       make(chan *types.KeyEvent, 128),
		cutTxtEvsQ:       make(chan *types.ClientCutText, 128),
		downKeys:         make([]uint32, 0),
		done:             make(chan struct{}),
	}
}

func (d *Display) GetDimensions() (width, height int) { return d.width, d.height }
func (d *Display) SetDimensions(width, height int)    { d.width, d.height = width, height }
func (d *Display) GetPixelFormat() *types.PixelFormat { return d.pixelFormat }
func (d *Display) SetPixelFormat(pf *types.PixelFormat) {
	d.pixelFormat = pf
}
func (d *Display) GetEncodings() []int32 { return d.encodings }
func (d *Display) SetEncodings(encs []int32, pseudoEns []int32) {
	d.encodings = encs
	d.pseudoEncodings = pseudoEns
	d.currentEnc = d.getEncodingsFunc(encs)
}

func (d *Display) GetCurrentEncoding() encodings.Encoding {
	if d.currentEnc != nil {
		return d.currentEnc
	}
	// Safe fallback order: Tight -> Raw
	return encodings.NewTight(encodings.TightOptions{JPEGQuality: 75})
}

// GetLastImage blocks until a frame is available (or provider closed).
func (d *Display) GetLastImage() *image.RGBA { return d.displayProvider.PullFrame() }

// Dispatch methods
func (d *Display) DispatchFrameBufferUpdate(req *types.FrameBufferUpdateRequest) { d.fbReqQueue <- req }
func (d *Display) DispatchKeyEvent(ev *types.KeyEvent)                           { d.keyEvQueue <- ev }
func (d *Display) DispatchPointerEvent(ev *types.PointerEvent)                   { d.ptrEvQueue <- ev }
func (d *Display) DispatchClientCutText(ev *types.ClientCutText)                 { d.cutTxtEvsQ <- ev }

// Start provider and watchers.
func (d *Display) Start() error {
	w, h := d.GetDimensions()
	if err := d.displayProvider.Start(w, h); err != nil {
		return err
	}
	go d.watchChannels()
	return nil
}

// Close stops everything and releases references so GC can collect promptly.
func (d *Display) Close() error {
	close(d.done)
	close(d.fbReqQueue)
	close(d.ptrEvQueue)
	close(d.keyEvQueue)
	close(d.cutTxtEvsQ)

	err := d.displayProvider.Close()
	d.displayProvider = nil

	// Help GC drop memory quickly.
	d.downKeys = nil
	d.outBuf = nil
	return err
}
