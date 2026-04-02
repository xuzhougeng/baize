package screentrace

import (
	"context"
	"fmt"

	"github.com/kbinani/screenshot"
)

type captureFunc func(context.Context) (Capture, error)

func capturePrimaryDisplay(context.Context) (Capture, error) {
	if screenshot.NumActiveDisplays() < 1 {
		return Capture{}, fmt.Errorf("no active display available")
	}
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return Capture{}, err
	}
	imageBytes, width, height, hash, err := prepareAnalysisJPEG(img, DefaultMaxImageDimension, DefaultJPEGQuality)
	if err != nil {
		return Capture{}, err
	}
	return Capture{
		DisplayIndex: 0,
		Width:        width,
		Height:       height,
		ImageBytes:   imageBytes,
		ImageHash:    hash,
	}, nil
}
