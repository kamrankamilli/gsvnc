package providers

import "image"

// A Display is an interface that can be implemented by different types of frame sources.
type Display interface {
	Start(width, height int) error
	PullFrame() *image.RGBA
	Close() error
}

// Provider is an enum used for selecting a display provider.
type Provider string

const (
	ProviderGstreamer     = "gstreamer"
	ProviderScreenCapture = "screencap"
)

// GetDisplayProvider returns the provider to use for the given RFB connection.
func GetDisplayProvider(p Provider) Display {
	switch p {
	case ProviderGstreamer:
		return &Gstreamer{}
	case ProviderScreenCapture:
		return &ScreenCapture{}
	default:
		return nil
	}
}
