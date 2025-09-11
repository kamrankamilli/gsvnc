package display

import (
	"time"

	"github.com/kamrankamilli/gsvnc/pkg/internal/log"
	"github.com/kamrankamilli/gsvnc/pkg/rfb/types"
)

func (d *Display) handleKeyEvents() {
	for {
		select {
		case <-d.done:
			return
		case ev, ok := <-d.keyEvQueue:
			if !ok {
				return
			}
			log.Debug("Got key event: ", ev)
			if ev.IsDown() {
				d.appendDownKeyIfMissing(ev.Key)
				d.dispatchDownKeys()
			} else {
				d.removeDownKey(ev.Key)
			}
		}
	}
}

func (d *Display) handlePointerEvents() {
	ticker := time.NewTicker(time.Millisecond * 8)
	defer ticker.Stop()

	var pending *types.PointerEvent
	for {
		select {
		case <-d.done:
			return
		case ev, ok := <-d.ptrEvQueue:
			if !ok {
				return
			}
			pending = ev
		case <-ticker.C:
			if pending != nil {
				d.servePointerEvent(pending)
				pending = nil
			}
		}
	}
}

func (d *Display) handleFrameBufferEvents() {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-d.done:
			return

		case ur, ok := <-d.fbReqQueue:
			if !ok {
				return
			}
			log.Debug("Handling framebuffer update request")
			d.pushFrame(ur)

		case <-ticker.C:
			// Only push keepalive when writer isn't busy and is open.
			if d.buf != nil {
				if d.buf.IsClosed() {
					return
				}
				if d.buf.Pending() > 0 {
					continue
				}
			}
			last := d.GetLastImage()
			if last != nil {
				d.pushImage(last)
			}
		}
	}
}

func (d *Display) handleCutTextEvents() {
	for {
		select {
		case <-d.done:
			return
		case ev, ok := <-d.cutTxtEvsQ:
			if !ok {
				return
			}
			log.Debug("Got cut-text event: ", ev)
			d.syncToClipboard(ev)
		}
	}
}

func (d *Display) watchChannels() {
	go d.handleKeyEvents()
	go d.handlePointerEvents()
	go d.handleFrameBufferEvents()
	go d.handleCutTextEvents()
}
