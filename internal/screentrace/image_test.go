package screentrace

import (
	"image"
	"image/color"
	"testing"
)

func TestDifferenceHashDistance(t *testing.T) {
	t.Parallel()

	left := image.NewRGBA(image.Rect(0, 0, 32, 32))
	fillRect(left, color.RGBA{255, 255, 255, 255})
	fillHalf(left, color.RGBA{0, 0, 0, 255})

	right := image.NewRGBA(image.Rect(0, 0, 32, 32))
	fillRect(right, color.RGBA{255, 255, 255, 255})
	fillHalf(right, color.RGBA{0, 0, 0, 255})

	other := image.NewRGBA(image.Rect(0, 0, 32, 32))
	fillRect(other, color.RGBA{0, 0, 0, 255})
	fillHalf(other, color.RGBA{255, 255, 255, 255})

	leftHash := differenceHashHex(left)
	rightHash := differenceHashHex(right)
	otherHash := differenceHashHex(other)

	if got := hashDistance(leftHash, rightHash); got != 0 {
		t.Fatalf("expected identical hash distance 0, got %d", got)
	}
	if got := hashDistance(leftHash, otherHash); got == 0 {
		t.Fatalf("expected different hashes, got %d", got)
	}
}

func TestPrepareAnalysisJPEGResizes(t *testing.T) {
	t.Parallel()

	src := image.NewRGBA(image.Rect(0, 0, 2400, 1200))
	fillRect(src, color.RGBA{120, 120, 120, 255})

	data, width, height, hash, err := prepareAnalysisJPEG(src, 800, 70)
	if err != nil {
		t.Fatalf("prepare analysis jpeg: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected jpeg data")
	}
	if width != 800 || height != 400 {
		t.Fatalf("unexpected resized dimensions: %d x %d", width, height)
	}
	if hash == "" {
		t.Fatal("expected non-empty image hash")
	}
}

func fillRect(img *image.RGBA, value color.RGBA) {
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			img.SetRGBA(x, y, value)
		}
	}
}

func fillHalf(img *image.RGBA, value color.RGBA) {
	mid := img.Bounds().Dx() / 2
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Min.X+mid; x++ {
			img.SetRGBA(x, y, value)
		}
	}
}
