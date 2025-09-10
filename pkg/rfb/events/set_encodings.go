package events

import (
	"github.com/kamrankamilli/gsvnc/pkg/buffer"
	"github.com/kamrankamilli/gsvnc/pkg/display"
	"github.com/kamrankamilli/gsvnc/pkg/internal/log"
)

// SetEncodings handles the client set-encodings event.
type SetEncodings struct{}

func (s *SetEncodings) Code() uint8 { return 2 }

func (s *SetEncodings) Handle(buf *buffer.ReadWriter, d *display.Display) error {
	if err := buf.ReadPadding(1); err != nil {
		return err
	}

	var numEncodings uint16
	if err := buf.Read(&numEncodings); err != nil {
		return err
	}
	encTypes := make([]int32, int(numEncodings))
	for i := 0; i < int(numEncodings); i++ {
		if err := buf.Read(&encTypes[i]); err != nil {
			return err
		}
	}

	encs, pseudo := splitPseudoEncodings(encTypes)
	log.Infof("Client encodings: %#v", encs)
	log.Infof("Client pseudo-encodings: %#v", pseudo)
	d.SetEncodings(encs, pseudo)
	return nil
}

func splitPseudoEncodings(encs []int32) (encodings, pseudoEncodings []int32) {
	encodings = make([]int32, 0, len(encs))
	i := 0
	for ; i < len(encs); i++ {
		encodings = append(encodings, encs[i])
		if encs[i] == 0 {
			break
		}
	}
	if i < len(encs)-1 {
		pseudoEncodings = encs[i+1:]
	}
	return
}
