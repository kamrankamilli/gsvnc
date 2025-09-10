package providers

import (
	"image"
	"image/draw"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/kamrankamilli/gsvnc/pkg/internal/log"
	"github.com/nfnt/resize"
)

// ScreenCapture implements a display provider that periodically captures the screen.
type ScreenCapture struct {
	frameQueue chan *image.RGBA
	stopCh     chan struct{}
	// reuse two buffers to avoid allocs
	workA *image.RGBA
	workB *image.RGBA
	swap  bool
}

func (s *ScreenCapture) Close() error {
	if s.stopCh != nil {
		close(s.stopCh)
	}
	return nil
}

func (s *ScreenCapture) PullFrame() *image.RGBA {
	select {
	case f := <-s.frameQueue:
		return f
	case <-s.stopCh:
		return nil
	}
}

func (s *ScreenCapture) Start(width, height int) error {
	s.frameQueue = make(chan *image.RGBA, 2)
	s.stopCh = make(chan struct{})
	s.workA = image.NewRGBA(image.Rect(0, 0, width, height))
	s.workB = image.NewRGBA(image.Rect(0, 0, width, height))

	go func() {
		ticker := time.NewTicker(200 * time.Millisecond) // ~5 FPS
		defer ticker.Stop()

		for {
			select {
			case <-s.stopCh:
				log.Debug("Stopping screen capture")
				return
			case <-ticker.C:
				bitMap := robotgo.CaptureScreen()
				if bitMap == nil {
					log.Error("CaptureScreen returned nil bitmap")
					continue
				}

				img := robotgo.ToImage(bitMap)
				robotgo.FreeBitmap(bitMap)

				if img == nil {
					log.Error("robotgo.ToImage returned nil image")
					continue
				}

				// Resize if needed
				b := img.Bounds()
				if b.Dx() != width || b.Dy() != height {
					img = resize.Resize(uint(width), uint(height), img, resize.NearestNeighbor)
				}

				// Choose work buffer
				dst := s.workA
				if s.swap {
					dst = s.workB
				}
				s.swap = !s.swap

				// Copy into RGBA buffer
				draw.Draw(dst, dst.Bounds(), img, img.Bounds().Min, draw.Src)

				// Non-blocking enqueue keeping only latest
				select {
				case s.frameQueue <- dst:
				default:
					select {
					case <-s.frameQueue:
					default:
					}
					select {
					case s.frameQueue <- dst:
					default:
					}
				}
			}
		}
	}()
	return nil
}
