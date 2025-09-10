package display

import (
	"math"

	"github.com/go-vgo/robotgo"
	"github.com/kamrankamilli/gsvnc/pkg/rfb/types"
)

func (d *Display) servePointerEvent(ev *types.PointerEvent) {
	// Scale remote coords to local screen if sizes differ
	x, y := int(ev.X), int(ev.Y)
	sw, sh := robotgo.GetScreenSize()
	if d.width > 0 && d.height > 0 && (d.width != sw || d.height != sh) {
		x = int(math.Round(float64(x) * float64(sw) / float64(d.width)))
		y = int(math.Round(float64(y) * float64(sh) / float64(d.height)))
	}

	// Move cursor first
	robotgo.Move(x, y)

	// Buttons (bits 0..2): left, middle, right — edge detect press/release
	btnNames := []string{"left", "middle", "right"}
	for i, name := range btnNames {
		prev := nthBitOf(d.lastBtnMask, i)
		cur := nthBitOf(ev.ButtonMask, i)
		if prev != cur {
			if cur == 1 {
				robotgo.MouseDown(name) // press
			} else {
				robotgo.MouseUp(name) // release
			}
		}
	}

	// Scroll (bits 3..6). robotgo.Scroll(x, y):
	//   y > 0 = scroll up, y < 0 = scroll down
	//   x > 0 = right,     x < 0 = left
	if nthBitOf(ev.ButtonMask, 3) == 1 { // up
		robotgo.Scroll(0, 1)
	}
	if nthBitOf(ev.ButtonMask, 4) == 1 { // down
		robotgo.Scroll(0, -1)
	}
	if nthBitOf(ev.ButtonMask, 5) == 1 { // left
		robotgo.Scroll(-1, 0)
	}
	if nthBitOf(ev.ButtonMask, 6) == 1 { // right
		robotgo.Scroll(1, 0)
	}

	// Save for next edge detection
	d.lastBtnMask = ev.ButtonMask
}

func nthBitOf(bit uint8, n int) uint8 { return (bit & (1 << n)) >> n }
