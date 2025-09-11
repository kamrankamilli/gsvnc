package display

import (
	"math"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/kamrankamilli/gsvnc/pkg/rfb/types"
)

var (
	cachedScreenW, cachedScreenH int
	lastScaleCheck               time.Time
	lastMouseX, lastMouseY       int
	lastMoveAt                   time.Time
)

func (d *Display) servePointerEvent(ev *types.PointerEvent) {
	// Recheck screen size at most every 2s
	now := time.Now()
	if now.Sub(lastScaleCheck) > 2*time.Second || cachedScreenW == 0 {
		cachedScreenW, cachedScreenH = robotgo.GetScreenSize()
		lastScaleCheck = now
	}

	x, y := int(ev.X), int(ev.Y)
	if d.width > 0 && d.height > 0 && (d.width != cachedScreenW || d.height != cachedScreenH) {
		x = int(math.Round(float64(x) * float64(cachedScreenW) / float64(d.width)))
		y = int(math.Round(float64(y) * float64(cachedScreenH) / float64(d.height)))
	}

	// Skip no-op move, and soft throttle to ~100–150 Hz
	if (x != lastMouseX || y != lastMouseY) && now.Sub(lastMoveAt) >= 7*time.Millisecond {
		robotgo.Move(x, y)
		lastMouseX, lastMouseY = x, y
		lastMoveAt = now
	}

	// Buttons (edge detection)
	btnNames := []string{"left", "middle", "right"}
	for i, name := range btnNames {
		prev := nthBitOf(d.lastBtnMask, i)
		cur := nthBitOf(ev.ButtonMask, i)
		if prev != cur {
			if cur == 1 {
				robotgo.MouseDown(name)
			} else {
				robotgo.MouseUp(name)
			}
		}
	}

	// Scroll — batch per event (still 1 unit per bit)
	if nthBitOf(ev.ButtonMask, 3) == 1 {
		robotgo.Scroll(0, 1)
	}
	if nthBitOf(ev.ButtonMask, 4) == 1 {
		robotgo.Scroll(0, -1)
	}
	if nthBitOf(ev.ButtonMask, 5) == 1 {
		robotgo.Scroll(-1, 0)
	}
	if nthBitOf(ev.ButtonMask, 6) == 1 {
		robotgo.Scroll(1, 0)
	}

	d.lastBtnMask = ev.ButtonMask
}

func nthBitOf(bit uint8, n int) uint8 { return (bit & (1 << n)) >> n }
