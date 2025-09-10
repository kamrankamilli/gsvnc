package encodings

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"log"
	"strconv"

	"github.com/kamrankamilli/gsvnc/pkg/internal/util"
	"github.com/kamrankamilli/gsvnc/pkg/rfb/types"
)

// TightPNGEncoding implements an Encoding interface using Tight encoding.
type TightPNGEncoding struct{}

// Code returns the code
func (t *TightPNGEncoding) Code() int32 { return -260 }

// HandleBuffer handles an image sample.
func (t *TightPNGEncoding) HandleBuffer(w io.Writer, f *types.PixelFormat, img *image.RGBA) {
	compressed := new(bytes.Buffer)

	if err := png.Encode(compressed, img); err != nil {
		log.Println("[tight-png] Could not encode image frame to png")
		return
	}

	buf := compressed.Bytes()

	i, _ := strconv.ParseInt("01010000", 2, 64) // PNG encoding
	_ = util.Write(w, uint8(i))
	_ = util.Write(w, computeTightLength(len(buf)))
	_ = util.Write(w, buf)
}
