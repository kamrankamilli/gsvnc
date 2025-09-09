package encodings

import (
	"image"
	"io"

	"github.com/kamrankamilli/gsvnc/pkg/rfb/types"
)

// RawEncoding implements an Encoding interface using raw pixel data.
type RawEncoding struct{}

// Code returns the code for RAW.
func (r *RawEncoding) Code() int32 { return 0 }

// HandleBuffer handles an image sample.
func (r *RawEncoding) HandleBuffer(w io.Writer, f *types.PixelFormat, img *image.RGBA) {
	_, _ = w.Write(applyPixelFormat(img, f))
}
