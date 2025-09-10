package display

import (
	"image"

	"github.com/kamrankamilli/gsvnc/pkg/buffer"
	"github.com/kamrankamilli/gsvnc/pkg/display/providers"
	"github.com/kamrankamilli/gsvnc/pkg/rfb/encodings"
	"github.com/kamrankamilli/gsvnc/pkg/rfb/types"
)

// Display represents a session with the local display. It manages the pipelines and events.
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

	// done closes when the Display is shutting down, so watchers can exit.
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

// GetDimensions returns the current dimensions of the display.
func (d *Display) GetDimensions() (width, height int) { return d.width, d.height }

// SetDimensions sets the dimensions of the display.
func (d *Display) SetDimensions(width, height int) { d.width, d.height = width, height }

// GetPixelFormat returns the current pixel format for the display.
func (d *Display) GetPixelFormat() *types.PixelFormat { return d.pixelFormat }

// SetPixelFormat sets the pixel format for the display.
func (d *Display) SetPixelFormat(pf *types.PixelFormat) { d.pixelFormat = pf }

// GetEncodings returns the encodings currently supported by the client.
func (d *Display) GetEncodings() []int32 { return d.encodings }

// SetEncodings sets the encodings that the connected client supports.
func (d *Display) SetEncodings(encs []int32, pseudoEns []int32) {
	d.encodings = encs
	d.pseudoEncodings = pseudoEns
	d.currentEnc = d.getEncodingsFunc(encs)
}

// GetCurrentEncoding returns the encoder in use.
func (d *Display) GetCurrentEncoding() encodings.Encoding { return d.currentEnc }

// GetLastImage returns the most recent frame for the display (blocks until available).
func (d *Display) GetLastImage() *image.RGBA { return d.displayProvider.PullFrame() }

// Dispatch methods
func (d *Display) DispatchFrameBufferUpdate(req *types.FrameBufferUpdateRequest) { d.fbReqQueue <- req }
func (d *Display) DispatchKeyEvent(ev *types.KeyEvent)                           { d.keyEvQueue <- ev }
func (d *Display) DispatchPointerEvent(ev *types.PointerEvent)                   { d.ptrEvQueue <- ev }
func (d *Display) DispatchClientCutText(ev *types.ClientCutText)                 { d.cutTxtEvsQ <- ev }

// Start underlying display provider.
func (d *Display) Start() error {
	w, h := d.GetDimensions()
	if err := d.displayProvider.Start(w, h); err != nil {
		return err
	}
	go d.watchChannels()
	return nil
}

// Close will stop the provider and close channels.
func (d *Display) Close() error {
	// Signal watchers to exit and close queues to end range loops
	close(d.done)
	close(d.fbReqQueue)
	close(d.ptrEvQueue)
	close(d.keyEvQueue)
	close(d.cutTxtEvsQ)
	return d.displayProvider.Close()
}
