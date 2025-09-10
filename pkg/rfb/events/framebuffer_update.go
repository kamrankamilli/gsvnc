package events

import (
	"github.com/kamrankamilli/gsvnc/pkg/buffer"
	"github.com/kamrankamilli/gsvnc/pkg/display"
	"github.com/kamrankamilli/gsvnc/pkg/rfb/types"
)

// FrameBufferUpdate handles framebuffer update events.
type FrameBufferUpdate struct{}

func (f *FrameBufferUpdate) Code() uint8 { return 3 }

func (f *FrameBufferUpdate) Handle(buf *buffer.ReadWriter, d *display.Display) error {
	var req types.FrameBufferUpdateRequest
	if err := buf.ReadInto(&req); err != nil {
		return err
	}
	d.DispatchFrameBufferUpdate(&req)
	return nil
}
