package events

import (
	"github.com/kamrankamilli/gsvnc/pkg/buffer"
	"github.com/kamrankamilli/gsvnc/pkg/display"
	"github.com/kamrankamilli/gsvnc/pkg/internal/log"
	"github.com/kamrankamilli/gsvnc/pkg/rfb/types"
)

// SetPixelFormat handles the client set-pixel-format event.
type SetPixelFormat struct{}

func (s *SetPixelFormat) Code() uint8 { return 0 }

func (s *SetPixelFormat) Handle(buf *buffer.ReadWriter, d *display.Display) error {
	if err := buf.ReadPadding(3); err != nil {
		return err
	}
	var pf types.PixelFormat
	if err := buf.ReadInto(&pf); err != nil {
		return err
	}
	log.Infof("Client wants pixel format: %#v", pf)
	d.SetPixelFormat(&pf)
	return nil
}
