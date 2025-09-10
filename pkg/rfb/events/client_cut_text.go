package events

import (
	"github.com/kamrankamilli/gsvnc/pkg/buffer"
	"github.com/kamrankamilli/gsvnc/pkg/display"
	"github.com/kamrankamilli/gsvnc/pkg/rfb/types"
)

// ClientCutText handles new text in the client's cut buffer.
type ClientCutText struct{}

func (c *ClientCutText) Code() uint8 { return 6 }

func (c *ClientCutText) Handle(buf *buffer.ReadWriter, d *display.Display) error {
	var req types.ClientCutText

	if err := buf.ReadPadding(3); err != nil {
		return err
	}
	if err := buf.Read(&req.Length); err != nil {
		return err
	}

	req.Text = make([]byte, req.Length)
	if err := buf.Read(&req.Text); err != nil {
		return err
	}

	d.DispatchClientCutText(&req)
	return nil
}
