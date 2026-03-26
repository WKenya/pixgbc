package app

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppConvertWritesArtifacts(t *testing.T) {
	inputPath := writeFixturePNG(t, "input.png", 24, 24, true)
	outputPath := filepath.Join(t.TempDir(), "out.png")
	previewPath := filepath.Join(t.TempDir(), "preview.png")

	var stdout, stderr bytes.Buffer
	app := New(&stdout, &stderr)

	code := app.Run(context.Background(), []string{
		"convert",
		"--input", inputPath,
		"--output", outputPath,
		"--preview-out", previewPath,
		"--mode", "relaxed",
		"--palette", "gbc-olive",
	})
	if code != 0 {
		t.Fatalf("Run(convert) = %d, stderr = %s", code, stderr.String())
	}

	assertPNGExists(t, outputPath)
	assertPNGExists(t, previewPath)
}

func TestAppConvertEmitReviewWritesBundle(t *testing.T) {
	inputPath := writeFixturePNG(t, "input.png", 32, 16, false)
	outputDir := t.TempDir()
	outputPath := filepath.Join(outputDir, "out.png")
	reviewRoot := filepath.Join(outputDir, "reviews")

	var stdout, stderr bytes.Buffer
	app := New(&stdout, &stderr)

	code := app.Run(context.Background(), []string{
		"convert",
		"--input", inputPath,
		"--output", outputPath,
		"--mode", "cgb-bg",
		"--debug",
		"--emit-review", reviewRoot,
	})
	if code != 0 {
		t.Fatalf("Run(convert --emit-review) = %d, stderr = %s", code, stderr.String())
	}

	reviewDir := parseReviewDir(t, stdout.String())
	for _, name := range []string{"final.png", "preview.png", "meta.json", "debug.png"} {
		if _, err := os.Stat(filepath.Join(reviewDir, name)); err != nil {
			t.Fatalf("review artifact %s missing: %v", name, err)
		}
	}
}

func TestAppInspectEmitsJSONReport(t *testing.T) {
	inputPath := writeFixturePNG(t, "inspect.png", 18, 18, true)

	var stdout, stderr bytes.Buffer
	app := New(&stdout, &stderr)

	code := app.Run(context.Background(), []string{
		"inspect",
		"--input", inputPath,
		"--json",
	})
	if code != 0 {
		t.Fatalf("Run(inspect) = %d, stderr = %s", code, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("inspect json decode error = %v\nstdout=%s", err, stdout.String())
	}
	for _, key := range []string{"source", "dominant_colors", "recommendations", "strict_mode_analysis"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("inspect output missing %q: %s", key, stdout.String())
		}
	}
}

func TestAppPaletteListPrintsPresets(t *testing.T) {
	var stdout, stderr bytes.Buffer
	app := New(&stdout, &stderr)

	code := app.Run(context.Background(), []string{"palette", "list"})
	if code != 0 {
		t.Fatalf("Run(palette list) = %d, stderr = %s", code, stderr.String())
	}

	body := stdout.String()
	for _, snippet := range []string{"gbc-olive", "dmg-gray", "#"} {
		if !strings.Contains(body, snippet) {
			t.Fatalf("palette list missing %q: %s", snippet, body)
		}
	}
}

func TestAppRootHelpListsCommands(t *testing.T) {
	var stdout, stderr bytes.Buffer
	app := New(&stdout, &stderr)

	code := app.Run(context.Background(), []string{"--help"})
	if code != 0 {
		t.Fatalf("Run(--help) = %d, stderr = %s", code, stderr.String())
	}

	body := stdout.String()
	for _, snippet := range []string{"convert", "inspect", "palette", "serve"} {
		if !strings.Contains(body, snippet) {
			t.Fatalf("help missing %q: %s", snippet, body)
		}
	}
}

func writeFixturePNG(t *testing.T, name string, width, height int, withAlpha bool) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, name)
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			a := uint8(0xFF)
			if withAlpha && (x+y)%7 == 0 {
				a = 0x60
			}
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8((x*9 + y*3) % 256),
				G: uint8((x*5 + y*11) % 256),
				B: uint8((x*13 + y*7) % 256),
				A: a,
			})
		}
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("os.Create(%q) error = %v", path, err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("png.Encode(%q) error = %v", path, err)
	}

	return path
}

func assertPNGExists(t *testing.T, path string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if _, err := png.Decode(bytes.NewReader(data)); err != nil {
		t.Fatalf("png.Decode(%q) error = %v", path, err)
	}
}

func parseReviewDir(t *testing.T, stdout string) string {
	t.Helper()

	for _, line := range strings.Split(stdout, "\n") {
		if strings.HasPrefix(line, "review_dir\t") {
			return strings.TrimPrefix(line, "review_dir\t")
		}
	}
	t.Fatalf("review_dir missing from stdout: %s", stdout)
	return ""
}
