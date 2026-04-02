package screencapture

import (
	"context"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefinitionIsComplete(t *testing.T) {
	t.Parallel()

	def := Definition().Normalized()
	if def.Name != ToolName {
		t.Fatalf("Definition().Name = %q, want %q", def.Name, ToolName)
	}
	fields := map[string]string{
		"FamilyKey":         def.FamilyKey,
		"FamilyTitle":       def.FamilyTitle,
		"DisplayTitle":      def.DisplayTitle,
		"Purpose":           def.Purpose,
		"Description":       def.Description,
		"InputContract":     def.InputContract,
		"OutputContract":    def.OutputContract,
		"Usage":             def.Usage,
		"InputJSONExample":  def.InputJSONExample,
		"OutputJSONExample": def.OutputJSONExample,
	}
	for name, value := range fields {
		if strings.TrimSpace(value) == "" {
			t.Fatalf("Definition().%s is empty", name)
		}
	}
}

func TestNormalizeInputAppliesDefaultsAndClamp(t *testing.T) {
	t.Parallel()

	got := NormalizeInput(ToolInput{
		MaxDimension: 1,
		JPEGQuality:  200,
	})
	if got.Analyze == nil || !*got.Analyze {
		t.Fatalf("NormalizeInput() analyze = %#v, want true", got.Analyze)
	}
	if got.MaxDimension != minMaxDim {
		t.Fatalf("NormalizeInput() max_dimension = %d, want %d", got.MaxDimension, minMaxDim)
	}
	if got.JPEGQuality != maxJPEGQuality {
		t.Fatalf("NormalizeInput() jpeg_quality = %d, want %d", got.JPEGQuality, maxJPEGQuality)
	}

	disable := false
	got = NormalizeInput(ToolInput{
		Analyze:      &disable,
		MaxDimension: 999999,
		JPEGQuality:  1,
	})
	if got.Analyze == nil || *got.Analyze {
		t.Fatalf("NormalizeInput() analyze = %#v, want false", got.Analyze)
	}
	if got.MaxDimension != maxMaxDim {
		t.Fatalf("NormalizeInput() max_dimension = %d, want %d", got.MaxDimension, maxMaxDim)
	}
	if got.JPEGQuality != minJPEGQuality {
		t.Fatalf("NormalizeInput() jpeg_quality = %d, want %d", got.JPEGQuality, minJPEGQuality)
	}
}

func TestEncodeResizedJPEGRespectsMaxDimension(t *testing.T) {
	t.Parallel()

	source := makeSampleImage(2400, 1200)
	captured, err := encodeResizedJPEG(3, source, NormalizeInput(ToolInput{
		MaxDimension: 1000,
		JPEGQuality:  80,
	}))
	if err != nil {
		t.Fatalf("encodeResizedJPEG() error = %v", err)
	}
	if captured.DisplayIndex != 3 {
		t.Fatalf("encodeResizedJPEG() display_index = %d, want 3", captured.DisplayIndex)
	}
	if captured.Width != 1000 || captured.Height != 500 {
		t.Fatalf("encodeResizedJPEG() size = %dx%d, want 1000x500", captured.Width, captured.Height)
	}
	if len(captured.JPEGBytes) == 0 {
		t.Fatal("encodeResizedJPEG() returned empty JPEG bytes")
	}
}

func TestExecuteWritesImageAndRunsAnalyzer(t *testing.T) {
	t.Parallel()

	capture := capturedImage{
		DisplayIndex: 0,
		Width:        1280,
		Height:       720,
		JPEGBytes:    mustJPEGBytes(t, 1280, 720),
	}
	now := time.Date(2026, 4, 2, 11, 30, 0, 123456789, time.UTC)

	var (
		gotFileName string
		gotImageURL string
	)
	result, err := Execute(context.Background(), ToolInput{}, ExecuteOptions{
		BaseDir: t.TempDir(),
		Now:     func() time.Time { return now },
		CaptureFn: func(context.Context, ToolInput) (capturedImage, error) {
			return capture, nil
		},
		Analyzer: func(_ context.Context, fileName, imageURL string) (string, error) {
			gotFileName = fileName
			gotImageURL = imageURL
			return "编辑器和终端同时可见。", nil
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	wantName := "screen-20260402-113000-123456789.jpg"
	if filepath.Base(result.Path) != wantName {
		t.Fatalf("Execute() file name = %q, want %q", filepath.Base(result.Path), wantName)
	}
	if gotFileName != wantName {
		t.Fatalf("Analyzer fileName = %q, want %q", gotFileName, wantName)
	}
	if !strings.HasPrefix(gotImageURL, "data:image/jpeg;base64,") {
		t.Fatalf("Analyzer imageURL = %q, want data URL prefix", gotImageURL)
	}
	if result.AnalysisStatus != "summarized" {
		t.Fatalf("Execute() analysis_status = %q, want summarized", result.AnalysisStatus)
	}
	if result.Summary != "编辑器和终端同时可见。" {
		t.Fatalf("Execute() summary = %q", result.Summary)
	}
	data, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", result.Path, err)
	}
	if len(data) == 0 {
		t.Fatalf("saved file %q is empty", result.Path)
	}
}

func TestExecuteCanSkipAnalysis(t *testing.T) {
	t.Parallel()

	analyze := false
	result, err := Execute(context.Background(), ToolInput{Analyze: &analyze}, ExecuteOptions{
		BaseDir: t.TempDir(),
		CaptureFn: func(context.Context, ToolInput) (capturedImage, error) {
			return capturedImage{
				DisplayIndex: 1,
				Width:        800,
				Height:       600,
				JPEGBytes:    mustJPEGBytes(t, 800, 600),
			}, nil
		},
		Analyzer: func(context.Context, string, string) (string, error) {
			t.Fatal("Analyzer should not be called when analyze=false")
			return "", nil
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.AnalysisStatus != "skipped" {
		t.Fatalf("Execute() analysis_status = %q, want skipped", result.AnalysisStatus)
	}
	if result.Summary != "" {
		t.Fatalf("Execute() summary = %q, want empty", result.Summary)
	}
}

func mustJPEGBytes(t *testing.T, width, height int) []byte {
	t.Helper()

	captured, err := encodeResizedJPEG(0, makeSampleImage(width, height), NormalizeInput(ToolInput{
		MaxDimension: width,
		JPEGQuality:  80,
	}))
	if err != nil {
		t.Fatalf("encodeResizedJPEG() error = %v", err)
	}
	return captured.JPEGBytes
}

func makeSampleImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(x % 255),
				G: uint8(y % 255),
				B: uint8((x + y) % 255),
				A: 255,
			})
		}
	}
	return img
}
