package display

import (
	"github.com/go-vgo/robotgo"
	"github.com/kamrankamilli/gsvnc/pkg/rfb/types"
)

func (d *Display) syncToClipboard(ev *types.ClientCutText) { robotgo.WriteAll(toUTF8(ev.Text)) }

func toUTF8(in []byte) string {
	// Treat bytes as Latin-1/ASCII fallback
	buf := make([]rune, len(in))
	for i, b := range in {
		buf[i] = rune(b)
	}
	return string(buf)
}
