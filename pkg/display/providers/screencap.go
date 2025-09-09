package providers

import (
	"image"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/kamrankamilli/gsvnc/pkg/internal/log"
	"github.com/nfnt/resize"
)

// ScreenCapture implements a display provider that periodically captures the screen
// using native APIs.
type ScreenCapture struct {
	frameQueue chan *image.RGBA // A channel that will essentially only ever have the latest frame available.
	stopCh     chan struct{}
}

// Close stops the gstreamer pipeline.
func (s *ScreenCapture) Close() error {
	s.stopCh <- struct{}{}
	return nil
}

// PullFrame returns a frame from the queue.
func (s *ScreenCapture) PullFrame() *image.RGBA { return <-s.frameQueue }

// Start starts the screen capture loop.
func (s *ScreenCapture) Start(width, height int) error {
	s.frameQueue = make(chan *image.RGBA, 2)
	s.stopCh = make(chan struct{})

	go func() {
		ticker := time.NewTicker(200 * time.Millisecond) // ~5 FPS
		defer ticker.Stop()

		for {
			select {
			case <-s.stopCh:
				log.Debug("Received event on stop channel, stopping screen capture")
				return
			case <-ticker.C:
				bitMap := robotgo.CaptureScreen()
				if bitMap == nil {
					log.Error("CaptureScreen returned nil bitmap")
					continue
				}
				// Always free the native bitmap.
				defer robotgo.FreeBitmap(bitMap)

				// Convert native bitmap directly to Go image.Image (no BMP decode needed).
				img := robotgo.ToImage(bitMap)
				if img == nil {
					log.Error("robotgo.ToImage returned nil image")
					continue
				}

				// Resize if larger than target dims (keeps your original behavior of forcing width/height).
				b := img.Bounds()
				if b.Max.X > width || b.Max.Y > height {
					img = resize.Resize(uint(width), uint(height), img, resize.Lanczos3)
				}

				// Ensure *image.RGBA for downstream consumers.
				var rgba *image.RGBA
				switch v := img.(type) {
				case *image.RGBA:
					rgba = v
				case *image.NRGBA:
					rgba = convertToRGBA(v)
				default:
					// Generic conversion for other image types.
					rect := img.Bounds()
					dst := image.NewRGBA(rect)
					// Avoid importing draw if you prefer: manual pixel copy
					for y := rect.Min.Y; y < rect.Max.Y; y++ {
						for x := rect.Min.X; x < rect.Max.X; x++ {
							dst.Set(x, y, img.At(x, y))
						}
					}
					rgba = dst
				}

				// Queue the image for processing (keep only the latest frame).
				log.Debug("Queueing frame for processing")
				select {
				case s.frameQueue <- rgba:
				default:
					// Queue full: drop oldest, then enqueue latest (non-blocking).
					select {
					case <-s.frameQueue:
					default:
					}
					select {
					case s.frameQueue <- rgba:
					default:
						// If we still can't push, just drop this frame.
						log.Debug("Client is behind on frames, could not push to channel")
					}
				}
			}
		}
	}()

	return nil
}

func convertToRGBA(in *image.NRGBA) *image.RGBA {
	size := in.Bounds().Size()
	rect := image.Rect(0, 0, size.X, size.Y)
	wImg := image.NewRGBA(rect)
	// loop though all the x
	for x := 0; x < size.X; x++ {
		// and now loop thorough all of this x's y
		for y := 0; y < size.Y; y++ {
			wImg.Set(x, y, in.At(x, y))
		}
	}
	return wImg
}
