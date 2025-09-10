package events

import (
	"github.com/kamrankamilli/gsvnc/pkg/buffer"
	"github.com/kamrankamilli/gsvnc/pkg/display"
)

// Event is an interface implemented by client message handlers.
type Event interface {
	Code() uint8
	Handle(buf *buffer.ReadWriter, d *display.Display) error
}

var DefaultEvents = []Event{
	&SetEncodings{},
	&SetPixelFormat{},
	&FrameBufferUpdate{},
	&KeyEvent{},
	&PointerEvent{},
	&ClientCutText{},
}

func GetDefaults() []Event {
	out := make([]Event, len(DefaultEvents))
	copy(out, DefaultEvents)
	return out
}

func CloseEventHandlers(hdlrs map[uint8]Event) {
	for _, ev := range hdlrs {
		closer, ok := ev.(interface{ Close() })
		if ok {
			closer.Close()
		}
	}
}
