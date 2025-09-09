package events

import (
	"github.com/kamrankamilli/gsvnc/pkg/buffer"
	"github.com/kamrankamilli/gsvnc/pkg/display"
	"github.com/kamrankamilli/gsvnc/pkg/rfb/types"
)

// KeyEvent handles key events.
type KeyEvent struct{}

// Code returns the code.
func (s *KeyEvent) Code() uint8 { return 4 }

// Handle handles the event.
func (s *KeyEvent) Handle(buf *buffer.ReadWriter, d *display.Display) error {
	var req types.KeyEvent

	if err := buf.Read(&req.DownFlag); err != nil {
		return err
	}
	buf.ReadPadding(2)
	if err := buf.Read(&req.Key); err != nil {
		return err
	}
	d.DispatchKeyEvent(&req)
	return nil
}
