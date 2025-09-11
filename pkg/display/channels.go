package display

import (
	"time"

	"github.com/kamrankamilli/gsvnc/pkg/internal/log"
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
	for {
		select {
		case <-d.done:
			return
		case ev, ok := <-d.ptrEvQueue:
			if !ok {
				return
			}
			// Drain to the most recent event to avoid lag/backlog
			for {
				select {
				case next := <-d.ptrEvQueue:
					ev = next
				default:
					goto handle
				}
			}
		handle:
			d.servePointerEvent(ev)
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
