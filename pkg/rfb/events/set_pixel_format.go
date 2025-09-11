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

	// Enforce true-colour; fall back if unsupported by our encoders
	if pf.TrueColour == 0 {
		log.Warning("Client requested non true-colour; keeping server default (16bpp 5-6-5).")
		// Keep existing display format (or set default if missing)
		if d.GetPixelFormat() == nil {
			d.SetPixelFormat(display.DefaultPixelFormat)
		}
		return nil
	}

	// Normalize/validate common fields we support
	// Allow 16 or 32 bpp; normalize others to 16 bpp 5-6-5
	if pf.BPP != 16 && pf.BPP != 32 {
		log.Warningf("Unsupported BPP=%d; using 16bpp 5-6-5 true-colour.", pf.BPP)
		d.SetPixelFormat(display.DefaultPixelFormat)
		return nil
	}
	if pf.BPP == 16 {
		// Ensure 5-6-5 layout
		pf.Depth = 16
		pf.BigEndian = 0
		pf.RedMax, pf.GreenMax, pf.BlueMax = 0x1f, 0x3f, 0x1f
		pf.RedShift, pf.GreenShift, pf.BlueShift = 11, 5, 0
	}
	// For 32bpp, most clients also send TrueColour with 8-8-8; many servers still encode JPEG fine.
	// We accept it as-is.

	d.SetPixelFormat(&pf)
	return nil
}
