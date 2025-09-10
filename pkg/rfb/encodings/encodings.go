package encodings

import (
	"image"
	"io"

	"github.com/kamrankamilli/gsvnc/pkg/rfb/types"
)

// Encoding is an interface to be implemented by different encoding handlers.
type Encoding interface {
	Code() int32
	HandleBuffer(w io.Writer, format *types.PixelFormat, img *image.RGBA)
}

// DefaultEncodings lists the encodings enabled by default on the server.
var DefaultEncodings = []Encoding{
	NewTight(TightOptions{JPEGQuality: 75}),
	&RawEncoding{}, // fallback if client doesn't speak Tight
}

// GetDefaults returns a slice of the default encoding handlers.
func GetDefaults() []Encoding {
	out := make([]Encoding, len(DefaultEncodings))
	for i, t := range DefaultEncodings {
		out[i] = t
	}
	return out
}
