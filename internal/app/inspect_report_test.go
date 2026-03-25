package app

import (
	"image"
	"image/color"
	"testing"

	"github.com/WKenya/pixgbc/internal/core"
)

func TestBuildInspectReportRecommendsCGBBGForTileFriendlyImage(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	fillInspectRect(img, image.Rect(0, 0, 8, 8), []color.NRGBA{
		{R: 0x10, G: 0x18, B: 0x10, A: 0xFF},
		{R: 0x38, G: 0x50, B: 0x30, A: 0xFF},
		{R: 0x78, G: 0x88, B: 0x40, A: 0xFF},
		{R: 0xD8, G: 0xE8, B: 0xA8, A: 0xFF},
	})
	fillInspectRect(img, image.Rect(8, 0, 16, 8), []color.NRGBA{
		{R: 0x18, G: 0x20, B: 0x18, A: 0xFF},
		{R: 0x40, G: 0x58, B: 0x38, A: 0xFF},
		{R: 0x80, G: 0x90, B: 0x48, A: 0xFF},
		{R: 0xD8, G: 0xE8, B: 0xA8, A: 0xFF},
	})
	fillInspectRect(img, image.Rect(0, 8, 8, 16), []color.NRGBA{
		{R: 0x18, G: 0x20, B: 0x18, A: 0xFF},
		{R: 0x40, G: 0x58, B: 0x38, A: 0xFF},
		{R: 0x80, G: 0x90, B: 0x48, A: 0xFF},
		{R: 0xD8, G: 0xE8, B: 0xA8, A: 0xFF},
	})
	fillInspectRect(img, image.Rect(8, 8, 16, 16), []color.NRGBA{
		{R: 0x20, G: 0x28, B: 0x20, A: 0xFF},
		{R: 0x48, G: 0x60, B: 0x40, A: 0xFF},
		{R: 0x88, G: 0x98, B: 0x50, A: 0xFF},
		{R: 0xD8, G: 0xE8, B: 0xA8, A: 0xFF},
	})

	report, err := buildInspectReport(img, core.SourceMeta{
		Width:      16,
		Height:     16,
		Format:     "png",
		FileSize:   128,
		FrameCount: 1,
	})
	if err != nil {
		t.Fatalf("buildInspectReport() error = %v", err)
	}

	if report.Recommendations.Mode != string(core.ModeCGBBG) {
		t.Fatalf("mode = %q, want %q", report.Recommendations.Mode, core.ModeCGBBG)
	}
	if report.Recommendations.PalettePreset == "" {
		t.Fatal("PalettePreset empty")
	}
	if !report.StrictModeAnalysis.Suitable {
		t.Fatal("StrictModeAnalysis.Suitable = false, want true")
	}
}

func TestBuildInspectReportRecommendsRelaxedForAlphaImage(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(x * 13),
				G: uint8(y * 11),
				B: uint8((x + y) * 7),
				A: uint8(80 + (x+y)%120),
			})
		}
	}

	report, err := buildInspectReport(img, core.SourceMeta{
		Width:      16,
		Height:     16,
		HasAlpha:   true,
		Format:     "png",
		FileSize:   256,
		FrameCount: 1,
	})
	if err != nil {
		t.Fatalf("buildInspectReport() error = %v", err)
	}

	if report.Recommendations.Mode != string(core.ModeRelaxed) {
		t.Fatalf("mode = %q, want %q", report.Recommendations.Mode, core.ModeRelaxed)
	}
	if report.StrictModeAnalysis.Suitable {
		t.Fatal("StrictModeAnalysis.Suitable = true, want false")
	}
}

func TestBuildInspectReportPrefersCuratedColorPresetOverGrayForColorfulImage(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	fillInspectRect(img, image.Rect(0, 0, 16, 16), []color.NRGBA{
		{R: 0x18, G: 0x2A, B: 0x18, A: 0xFF},
		{R: 0x3A, G: 0x58, B: 0x34, A: 0xFF},
		{R: 0x78, G: 0x8C, B: 0x40, A: 0xFF},
		{R: 0xD0, G: 0xD8, B: 0x7A, A: 0xFF},
	})

	report, err := buildInspectReport(img, core.SourceMeta{
		Width:      16,
		Height:     16,
		Format:     "png",
		FileSize:   128,
		FrameCount: 1,
	})
	if err != nil {
		t.Fatalf("buildInspectReport() error = %v", err)
	}

	if report.Recommendations.PalettePreset == "dmg-gray" {
		t.Fatalf("PalettePreset = %q, want non-gray curated preset", report.Recommendations.PalettePreset)
	}
}

func fillInspectRect(img *image.NRGBA, rect image.Rectangle, ramp []color.NRGBA) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetNRGBA(x, y, ramp[(x+y)%len(ramp)])
		}
	}
}
