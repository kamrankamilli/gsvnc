package providers

import (
	"fmt"
	"image"
	"io"
	"runtime"
	"sync"

	"github.com/go-gst/go-gst/gst"
	"github.com/go-gst/go-gst/gst/app"
	"github.com/go-gst/go-gst/gst/video"
	"github.com/kamrankamilli/gsvnc/pkg/internal/log"
)

// Gstreamer implements a display provider using gstreamer to capture video.
type Gstreamer struct {
	pipeline   *gst.Pipeline
	frameQueue chan *image.RGBA // latest-only queue

	// reuse two RGBA buffers to avoid per-frame allocs
	workA *image.RGBA
	workB *image.RGBA
	swap  bool
	w     int
	h     int

	linkMu     sync.Mutex
	linkedOnce bool

	done chan struct{} // signals Close to appsink/PullFrame
}

// Close stops the gstreamer pipeline and releases resources.
func (g *Gstreamer) Close() error {
	if g.done != nil {
		close(g.done)
	}
	if g.pipeline == nil {
		return nil
	}
	err := g.pipeline.SetState(gst.StateNull)
	g.pipeline.Unref()
	g.pipeline = nil
	return err
}

// PullFrame returns a frame or nil if closed.
func (g *Gstreamer) PullFrame() *image.RGBA {
	select {
	case f := <-g.frameQueue:
		return f
	case <-g.done:
		return nil
	}
}

// Start will start the gstreamer pipeline and send images to the frame queue.
func (g *Gstreamer) Start(width, height int) error {
	log.Debug("Building gstreamer pipeline for display connection")
	g.frameQueue = make(chan *image.RGBA, 2)
	g.w, g.h = width, height
	g.workA = image.NewRGBA(image.Rect(0, 0, width, height))
	g.workB = image.NewRGBA(image.Rect(0, 0, width, height))
	g.linkedOnce = false
	g.done = make(chan struct{})

	pipeline, err := gst.NewPipeline("")
	if err != nil {
		return err
	}

	// Get the screen capture element depending on the OS
	src, err := getScreenCaptureElement()
	if err != nil {
		return err
	}

	// Let decodebin decide best path
	decodebin, err := gst.NewElement("decodebin")
	if err != nil {
		return err
	}
	if err := pipeline.AddMany(src, decodebin); err != nil {
		return err
	}
	if err := src.Link(decodebin); err != nil {
		return fmt.Errorf("failed to link src->decodebin: %v", err)
	}

	// Build remaining pipeline once decodebin pads appear
	decodebin.Connect("pad-added", func(self *gst.Element, srcPad *gst.Pad) {
		g.linkMu.Lock()
		if g.linkedOnce {
			g.linkMu.Unlock()
			return
		}
		g.linkedOnce = true
		g.linkMu.Unlock()

		log.Debug("Decodebin pad added, linking pipeline")
		// queue ! videorate ! videoscale ! videoconvert ! capsfilter (RGBA WxH 5fps) ! appsink
		elements, err := gst.NewElementMany("queue", "videorate", "videoscale", "videoconvert", "capsfilter", "appsink")
		if err != nil {
			logPipelineErr(err)
			return
		}
		queue, videorate, videoscale, videoconvert, capsfilter, appsink :=
			elements[0], elements[1], elements[2], elements[3], elements[4], elements[5]

		// Caps
		rateCaps := gst.NewCapsFromString("video/x-raw,framerate=5/1")
		scaleCaps := gst.NewCapsFromString(fmt.Sprintf("video/x-raw,width=%d,height=%d", g.w, g.h))
		rgbaCaps := gst.NewCapsFromString(fmt.Sprintf("video/x-raw,format=RGBA,width=%d,height=%d,framerate=5/1", g.w, g.h))

		// Also describe to appsink via video.Info
		videoInfo := video.NewInfo().
			WithFormat(video.FormatRGBA, uint(g.w), uint(g.h)).
			WithFPS(gst.Fraction(5, 1))

		// Configure and link elements
		if err := runAllUntilError([]func() error{
			func() error { return videoscale.SetProperty("method", 0) }, // nearest neighbor (cheap)
			func() error { return pipeline.AddMany(elements...) },
			func() error { return queue.Link(videorate) },
			func() error { return videorate.LinkFiltered(videoscale, rateCaps) },
			func() error { return videoscale.LinkFiltered(videoconvert, scaleCaps) },
			func() error { capsfilter.SetProperty("caps", rgbaCaps); return nil },
			func() error { return videoconvert.Link(capsfilter) },
			func() error {
				sink := app.SinkFromElement(appsink)
				if sink == nil {
					return fmt.Errorf("appsink type assertion failed")
				}
				// Instruct appsink about caps; also drop when behind
				sink.SetCaps(videoInfo.ToCaps())
				sink.SetMaxBuffers(2)
				sink.SetDrop(true)
				// Pull samples via callbacks
				sink.SetCallbacks(&app.SinkCallbacks{
					NewSampleFunc: func(self *app.Sink) gst.FlowReturn {
						select {
						case <-g.done:
							return gst.FlowEOS
						default:
						}
						sample := self.PullSample()
						if sample == nil {
							return gst.FlowOK
						}
						defer sample.Unref()

						// Reader over mapped buffer
						r := sample.GetBuffer().Reader()
						if r == nil {
							return gst.FlowOK
						}

						// Choose reusable destination
						dst := g.workA
						if g.swap {
							dst = g.workB
						}
						g.swap = !g.swap

						// Expect tightly-packed RGBA
						need := g.w * g.h * 4
						n, err := io.ReadFull(r, dst.Pix[:need])
						if err != nil || n != need {
							logPipelineErr(fmt.Errorf("read RGBA bytes failed/short"))
							return gst.FlowError
						}

						// Non-blocking enqueue (keep latest)
						select {
						case <-g.done:
							return gst.FlowEOS
						case g.frameQueue <- dst:
						default:
							select {
							case <-g.frameQueue:
							default:
							}
							select {
							case <-g.done:
								return gst.FlowEOS
							case g.frameQueue <- dst:
							default:
							}
						}
						return gst.FlowOK
					},
				})
				return nil
			},
		}); err != nil {
			logPipelineErr(err)
			return
		}

		// Sync states and connect pad
		for _, e := range elements {
			if ok := e.SyncStateWithParent(); !ok {
				logPipelineErr(fmt.Errorf("could not sync element: %s", e.GetName()))
				return
			}
		}
		if ret := srcPad.Link(queue.GetStaticPad("sink")); ret != gst.PadLinkOK {
			log.Error("Could not link src pad to pipeline")
		}
	})

	// Intentionally no bus logger goroutine to avoid a stuck TimedPop after shutdown.

	g.pipeline = pipeline
	if err := pipeline.SetState(gst.StatePlaying); err != nil {
		_ = g.Close()
		return err
	}
	return nil
}

func getScreenCaptureElement() (elem *gst.Element, err error) {
	switch runtime.GOOS {
	case "windows":
		log.Debug("Detected Windows, using gdiscreencapsrc")
		elem, err = gst.NewElement("gdiscreencapsrc")
		if err == nil {
			_ = elem.SetProperty("cursor", true)
		}
	case "darwin":
		log.Debug("Detected macOS, using avfvideosrc")
		elem, err = gst.NewElement("avfvideosrc")
		if err == nil {
			if err = elem.SetProperty("capture-screen", true); err == nil {
				_ = elem.SetProperty("capture-screen-cursor", true)
			}
		}
	default:
		log.Debug("Detected Linux, using ximagesrc")
		elem, err = gst.NewElement("ximagesrc")
		if err == nil {
			_ = elem.SetProperty("show-pointer", true)
			_ = elem.SetProperty("use-damage", false)
		}
	}
	return
}

func runAllUntilError(fs []func() error) error {
	for _, f := range fs {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}

func logPipelineErr(err error) {
	log.Error("[go-gst-error] ", err.Error())
}
