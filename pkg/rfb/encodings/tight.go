package encodings

import (
	"bytes"
	"image"
	"image/jpeg"
	"io"
	"sync"

	"github.com/kamrankamilli/gsvnc/pkg/internal/util"
	"github.com/kamrankamilli/gsvnc/pkg/rfb/types"
)

// TightOptions lets you tune JPEG compression quality (1..100).
type TightOptions struct {
	JPEGQuality int
}

// TightEncoding implements Tight with JPEG compression.
type TightEncoding struct {
	quality int
	pool    *sync.Pool // bytes.Buffer pool to reduce GC pressure
}

// NewTight constructs a Tight encoder with options.
func NewTight(opts TightOptions) *TightEncoding {
	q := opts.JPEGQuality
	if q <= 0 {
		q = 75
	}
	if q > 100 {
		q = 100
	}
	return &TightEncoding{
		quality: q,
		pool: &sync.Pool{
			New: func() any { return new(bytes.Buffer) },
		},
	}
}

// Code returns the RFB encoding code for Tight.
func (t *TightEncoding) Code() int32 { return 7 }

// HandleBuffer JPEG-encodes the RGBA frame and writes Tight payload.
// Layout: [control byte=0x90] [varlen length] [JPEG bytes]
func (t *TightEncoding) HandleBuffer(w io.Writer, f *types.PixelFormat, img *image.RGBA) {
	// Encode to JPEG into pooled buffer
	buf := t.pool.Get().(*bytes.Buffer)
	buf.Reset()
	defer t.pool.Put(buf)

	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: t.quality}); err != nil {
		// drop frame if encode fails
		return
	}
	jpegBytes := buf.Bytes()

	// Control byte: 0b1001_0000 (JPEG basic)
	const tightJPEGCtrl = 0x90
	_ = util.Write(w, uint8(tightJPEGCtrl))
	_ = util.Write(w, computeTightLength(len(jpegBytes)))
	_ = util.Write(w, jpegBytes)
}

func computeTightLength(n int) []byte {
	// Tight length uses 1-3 bytes, 7 bits per byte with MSB as "more" flag.
	out := []byte{byte(n & 0x7F)}
	if n > 0x7F {
		out[0] |= 0x80
		out = append(out, byte((n>>7)&0x7F))
		if n > 0x3FFF {
			out[1] |= 0x80
			out = append(out, byte((n>>14)&0xFF))
		}
	}
	return out
}
