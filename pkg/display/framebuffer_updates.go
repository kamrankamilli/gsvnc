package display

import (
	"bytes"
	"image"
	"image/draw"
	"sync"

	"github.com/kamrankamilli/gsvnc/pkg/internal/log"
	"github.com/kamrankamilli/gsvnc/pkg/internal/util"
	"github.com/kamrankamilli/gsvnc/pkg/rfb/encodings"
	"github.com/kamrankamilli/gsvnc/pkg/rfb/types"
)

var fbBufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

const (
	encodingCopyRect     = 1
	cmdFramebufferUpdate = 0
)

func (d *Display) pushFrame(ur *types.FrameBufferUpdateRequest) {
	li := d.GetLastImage()
	if li == nil {
		return
	}
	if ur.Incremental() {
		li = truncateImage(ur, li)
	}
	if li == nil || li.Bounds().Empty() {
		return
	}
	log.Debug("Pushing frame to client")
	d.pushImage(li)
}

func (d *Display) pushImage(img *image.RGBA) {
	if img == nil || img.Bounds().Empty() {
		return
	}
	// If the writer is closed, drop immediately.
	if d.buf != nil && d.buf.IsClosed() {
		return
	}

	b := img.Bounds()
	format := d.GetPixelFormat()
	if format.TrueColour == 0 {
		// Fallback to a known-good format to keep the session alive
		format = DefaultPixelFormat
		d.SetPixelFormat(format)
	}
	enc := d.GetCurrentEncoding()
	if enc == nil {
		enc = &encodings.RawEncoding{}
	}

	// Use pooled buffer to reduce allocations
	buf := fbBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer fbBufPool.Put(buf)

	// header
	util.Write(buf, uint8(cmdFramebufferUpdate))
	util.Write(buf, uint8(0))  // padding
	util.Write(buf, uint16(1)) // rectangles=1

	// rectangle header
	util.PackStruct(buf, &types.FrameBufferRectangle{
		X:       uint16(b.Min.X),
		Y:       uint16(b.Min.Y),
		Width:   uint16(b.Dx()),
		Height:  uint16(b.Dy()),
		EncType: enc.Code(),
	})

	// payload by encoder
	enc.HandleBuffer(buf, d.GetPixelFormat(), img)

	// Final guard: drop if closed
	if d.buf != nil && d.buf.IsClosed() {
		return
	}
	// Keep only latest framebuffer in the queue (avoid backlog/latency).
	d.buf.DispatchLatest(buf.Bytes())
}

func truncateImage(ur *types.FrameBufferUpdateRequest, img *image.RGBA) *image.RGBA {
	r := image.Rect(
		int(ur.X),
		int(ur.Y),
		int(ur.X)+int(ur.Width),
		int(ur.Y)+int(ur.Height),
	).Intersect(img.Bounds())

	if r.Empty() {
		return image.NewRGBA(image.Rect(0, 0, 0, 0))
	}

	out := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(out, out.Bounds(), img, r.Min, draw.Src)
	return out
}
