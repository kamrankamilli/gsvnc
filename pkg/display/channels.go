package display

import (
	"time"

	"github.com/kamrankamilli/gsvnc/pkg/internal/log"
)

func (d *Display) handleKeyEvents() {
	for ev := range d.keyEvQueue {
		log.Debug("Got key event: ", ev)
		if ev.IsDown() {
			d.appendDownKeyIfMissing(ev.Key)
			d.dispatchDownKeys()
		} else {
			d.removeDownKey(ev.Key)
		}
	}
}

func (d *Display) handlePointerEvents() {
	for ev := range d.ptrEvQueue {
		log.Debug("Got pointer event: ", ev)
		d.servePointerEvent(ev)
	}
}

func (d *Display) handleFrameBufferEvents() {
	// Keep behavior but reduce busy pushing by aligning tick to provider FPS (~200ms).
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case ur, ok := <-d.fbReqQueue:
			if !ok {
				return
			}
			log.Debug("Handling framebuffer update request")
			d.pushFrame(ur)
		case <-ticker.C:
			// Periodic push of the latest frame (keep-alive / clients w/o frequent requests)
			log.Debug("Pushing latest frame to client (periodic)")
			last := d.GetLastImage()
			if last != nil {
				d.pushImage(last)
			}
		}
	}
}

func (d *Display) handleCutTextEvents() {
	for ev := range d.cutTxtEvsQ {
		log.Debug("Got cut-text event: ", ev)
		d.syncToClipboard(ev)
	}
}

func (d *Display) watchChannels() {
	go d.handleKeyEvents()
	go d.handlePointerEvents()
	go d.handleFrameBufferEvents()
	go d.handleCutTextEvents()
}
