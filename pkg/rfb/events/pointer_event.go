package events

import (
	"github.com/kamrankamilli/gsvnc/pkg/buffer"
	"github.com/kamrankamilli/gsvnc/pkg/display"
	"github.com/kamrankamilli/gsvnc/pkg/rfb/types"
)

type PointerEvent struct{}

func (s *PointerEvent) Code() uint8 { return 5 }

func (s *PointerEvent) Handle(buf *buffer.ReadWriter, d *display.Display) error {
	var req types.PointerEvent
	if err := buf.ReadInto(&req); err != nil {
		return err
	}
	d.DispatchPointerEvent(&req)
	return nil
}
