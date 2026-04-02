package screentrace

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"strings"
)

func prepareAnalysisJPEG(src image.Image, maxDimension, quality int) ([]byte, int, int, string, error) {
	if maxDimension <= 0 {
		maxDimension = DefaultMaxImageDimension
	}
	if quality <= 0 {
		quality = DefaultJPEGQuality
	}
	scaled := resizeToFit(src, maxDimension)
	bounds := scaled.Bounds()

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, scaled, &jpeg.Options{Quality: quality}); err != nil {
		return nil, 0, 0, "", err
	}
	hash := differenceHashHex(scaled)
	return buf.Bytes(), bounds.Dx(), bounds.Dy(), hash, nil
}

func jpegDataURL(data []byte) string {
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(data)
}

func resizeToFit(src image.Image, maxDimension int) image.Image {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return src
	}
	if width <= maxDimension && height <= maxDimension {
		return src
	}

	var dstWidth, dstHeight int
	if width >= height {
		dstWidth = maxDimension
		dstHeight = int(float64(height) * (float64(maxDimension) / float64(width)))
	} else {
		dstHeight = maxDimension
		dstWidth = int(float64(width) * (float64(maxDimension) / float64(height)))
	}
	if dstWidth < 1 {
		dstWidth = 1
	}
	if dstHeight < 1 {
		dstHeight = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, dstWidth, dstHeight))
	for y := 0; y < dstHeight; y++ {
		srcY := bounds.Min.Y + y*height/dstHeight
		for x := 0; x < dstWidth; x++ {
			srcX := bounds.Min.X + x*width/dstWidth
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}
	return dst
}

func differenceHashHex(src image.Image) string {
	scaled := image.NewGray(image.Rect(0, 0, 9, 8))
	bounds := src.Bounds()
	width := max(1, bounds.Dx())
	height := max(1, bounds.Dy())
	for y := 0; y < 8; y++ {
		srcY := bounds.Min.Y + y*height/8
		for x := 0; x < 9; x++ {
			srcX := bounds.Min.X + x*width/9
			scaled.SetGray(x, y, color.Gray{Y: luminance(src.At(srcX, srcY))})
		}
	}

	var hash uint64
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			left := scaled.GrayAt(x, y).Y
			right := scaled.GrayAt(x+1, y).Y
			hash <<= 1
			if left > right {
				hash |= 1
			}
		}
	}
	return fmt.Sprintf("%016x", hash)
}

func hashDistance(a, b string) int {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if len(a) != 16 || len(b) != 16 {
		if a == b {
			return 0
		}
		return 64
	}
	var distance int
	for i := 0; i < 16; i++ {
		xor := hexNibble(a[i]) ^ hexNibble(b[i])
		distance += bitsSet4(xor)
	}
	return distance
}

func hexNibble(b byte) byte {
	switch {
	case b >= '0' && b <= '9':
		return b - '0'
	case b >= 'a' && b <= 'f':
		return 10 + b - 'a'
	case b >= 'A' && b <= 'F':
		return 10 + b - 'A'
	default:
		return 0
	}
}

func bitsSet4(v byte) int {
	switch v & 0x0f {
	case 0, 1, 2, 4, 8:
		return 1 * boolToInt(v != 0)
	case 3, 5, 6, 9, 10, 12:
		return 2
	case 7, 11, 13, 14:
		return 3
	case 15:
		return 4
	default:
		return 0
	}
}

func luminance(c color.Color) uint8 {
	r, g, b, _ := c.RGBA()
	y := (299*r + 587*g + 114*b + 500) / 1000
	return uint8(y >> 8)
}
