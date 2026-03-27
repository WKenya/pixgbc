package render

import (
	"context"
	"image"
	"image/color"
	"slices"
	"testing"

	"github.com/WKenya/pixgbc/internal/core"
	"github.com/WKenya/pixgbc/internal/palette"
	"github.com/WKenya/pixgbc/internal/source"
)

func TestRunCGBBGProducesAssignmentsBanksAndDebug(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 8))
	fillRect(img, image.Rect(0, 0, 8, 8), []color.NRGBA{
		{R: 0x08, G: 0x10, B: 0x08, A: 0xFF},
		{R: 0x28, G: 0x38, B: 0x28, A: 0xFF},
		{R: 0x70, G: 0x80, B: 0x50, A: 0xFF},
		{R: 0xD8, G: 0xE8, B: 0xA8, A: 0xFF},
	})
	fillRect(img, image.Rect(8, 0, 16, 8), []color.NRGBA{
		{R: 0x20, G: 0x10, B: 0x10, A: 0xFF},
		{R: 0x58, G: 0x28, B: 0x20, A: 0xFF},
		{R: 0xA0, G: 0x60, B: 0x38, A: 0xFF},
		{R: 0xF0, G: 0xC0, B: 0x90, A: 0xFF},
	})

	result, err := RunCGBBG(context.Background(), source.NewSingleImage(img, core.SourceMeta{
		Width:      16,
		Height:     8,
		Format:     "png",
		FileSize:   64,
		FrameCount: 1,
	}), core.Config{
		Mode:            core.ModeCGBBG,
		TargetWidth:     16,
		TargetHeight:    8,
		TileSize:        8,
		ColorsPerTile:   4,
		MaxPalettes:     2,
		PreviewScale:    1,
		Dither:          core.DitherNone,
		PaletteStrategy: core.PaletteExtract,
		EmitDebug:       true,
	})
	if err != nil {
		t.Fatalf("RunCGBBG() error = %v", err)
	}

	if got := len(result.PaletteBanks); got != 2 {
		t.Fatalf("len(PaletteBanks) = %d, want 2", got)
	}
	if got := len(result.TileAssignments); got != 2 {
		t.Fatalf("len(TileAssignments) = %d, want 2", got)
	}
	if result.TileAssignments[0].PaletteBank == result.TileAssignments[1].PaletteBank {
		t.Fatalf("tile assignments should differ: %#v", result.TileAssignments)
	}
	if got := result.Metadata["tile_grid_width"]; got != 2 {
		t.Fatalf("tile_grid_width = %#v, want 2", got)
	}
	if got := result.Metadata["tile_grid_height"]; got != 1 {
		t.Fatalf("tile_grid_height = %#v, want 1", got)
	}
	if _, ok := result.DebugImages["tile-bank-heatmap"]; !ok {
		t.Fatal("missing tile-bank-heatmap debug image")
	}
}

func TestRunCGBBGMergesTilePalettesDownToMaxBanks(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 24, 8))
	fillRect(img, image.Rect(0, 0, 8, 8), []color.NRGBA{
		{R: 0x10, G: 0x10, B: 0x10, A: 0xFF},
		{R: 0x30, G: 0x30, B: 0x30, A: 0xFF},
		{R: 0x70, G: 0x70, B: 0x70, A: 0xFF},
		{R: 0xE0, G: 0xE0, B: 0xE0, A: 0xFF},
	})
	fillRect(img, image.Rect(8, 0, 16, 8), []color.NRGBA{
		{R: 0x18, G: 0x18, B: 0x18, A: 0xFF},
		{R: 0x38, G: 0x38, B: 0x38, A: 0xFF},
		{R: 0x78, G: 0x78, B: 0x78, A: 0xFF},
		{R: 0xE8, G: 0xE8, B: 0xE8, A: 0xFF},
	})
	fillRect(img, image.Rect(16, 0, 24, 8), []color.NRGBA{
		{R: 0x20, G: 0x40, B: 0x10, A: 0xFF},
		{R: 0x50, G: 0x80, B: 0x20, A: 0xFF},
		{R: 0x90, G: 0xB0, B: 0x50, A: 0xFF},
		{R: 0xE0, G: 0xF0, B: 0xA0, A: 0xFF},
	})

	result, err := RunCGBBG(context.Background(), source.NewSingleImage(img, core.SourceMeta{
		Width:      24,
		Height:     8,
		Format:     "png",
		FileSize:   96,
		FrameCount: 1,
	}), core.Config{
		Mode:            core.ModeCGBBG,
		TargetWidth:     24,
		TargetHeight:    8,
		TileSize:        8,
		ColorsPerTile:   4,
		MaxPalettes:     2,
		PreviewScale:    1,
		Dither:          core.DitherNone,
		PaletteStrategy: core.PaletteExtract,
	})
	if err != nil {
		t.Fatalf("RunCGBBG() error = %v", err)
	}

	if got := len(result.PaletteBanks); got != 2 {
		t.Fatalf("len(PaletteBanks) = %d, want 2", got)
	}
	if got := len(result.TileAssignments); got != 3 {
		t.Fatalf("len(TileAssignments) = %d, want 3", got)
	}
}

func TestRunCGBBGPresetModeConstrainsBanksToPreset(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 8))
	fillRect(img, image.Rect(0, 0, 8, 8), []color.NRGBA{
		{R: 0x10, G: 0x40, B: 0xB0, A: 0xFF},
		{R: 0x20, G: 0x60, B: 0xD0, A: 0xFF},
		{R: 0x40, G: 0x90, B: 0xF0, A: 0xFF},
		{R: 0xA0, G: 0xD0, B: 0xFF, A: 0xFF},
	})
	fillRect(img, image.Rect(8, 0, 16, 8), []color.NRGBA{
		{R: 0xB0, G: 0x30, B: 0x20, A: 0xFF},
		{R: 0xD0, G: 0x60, B: 0x30, A: 0xFF},
		{R: 0xF0, G: 0x90, B: 0x40, A: 0xFF},
		{R: 0xFF, G: 0xD0, B: 0x90, A: 0xFF},
	})

	result, err := RunCGBBG(context.Background(), source.NewSingleImage(img, core.SourceMeta{
		Width:      16,
		Height:     8,
		Format:     "png",
		FileSize:   64,
		FrameCount: 1,
	}), core.Config{
		Mode:            core.ModeCGBBG,
		TargetWidth:     16,
		TargetHeight:    8,
		TileSize:        8,
		ColorsPerTile:   4,
		MaxPalettes:     2,
		PreviewScale:    1,
		Dither:          core.DitherNone,
		PaletteStrategy: core.PalettePreset,
		PalettePreset:   "gbc-olive",
	})
	if err != nil {
		t.Fatalf("RunCGBBG() error = %v", err)
	}

	preset := palette.MustGetPreset("gbc-olive")
	for _, bank := range result.PaletteBanks {
		for _, c := range bank.Colors {
			if !slices.Contains(preset.Colors, c) {
				t.Fatalf("bank color %#v not in preset %#v", c, preset.Colors)
			}
		}
	}
}

func TestRunCGBBGExtractModeCanUseNonPresetColors(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	fillRect(img, img.Bounds(), []color.NRGBA{
		{R: 0x10, G: 0x40, B: 0xB0, A: 0xFF},
		{R: 0x20, G: 0x60, B: 0xD0, A: 0xFF},
		{R: 0x40, G: 0x90, B: 0xF0, A: 0xFF},
		{R: 0xA0, G: 0xD0, B: 0xFF, A: 0xFF},
	})

	result, err := RunCGBBG(context.Background(), source.NewSingleImage(img, core.SourceMeta{
		Width:      8,
		Height:     8,
		Format:     "png",
		FileSize:   64,
		FrameCount: 1,
	}), core.Config{
		Mode:            core.ModeCGBBG,
		TargetWidth:     8,
		TargetHeight:    8,
		TileSize:        8,
		ColorsPerTile:   4,
		MaxPalettes:     1,
		PreviewScale:    1,
		Dither:          core.DitherNone,
		PaletteStrategy: core.PaletteExtract,
		PalettePreset:   "gbc-olive",
	})
	if err != nil {
		t.Fatalf("RunCGBBG() error = %v", err)
	}

	preset := palette.MustGetPreset("gbc-olive")
	foundNonPreset := false
	for _, bank := range result.PaletteBanks {
		for _, c := range bank.Colors {
			if !slices.Contains(preset.Colors, c) {
				foundNonPreset = true
			}
		}
	}
	if !foundNonPreset {
		t.Fatalf("expected extract mode to use non-preset colors, got %#v", result.PaletteBanks)
	}
}

func fillRect(img *image.NRGBA, rect image.Rectangle, ramp []color.NRGBA) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetNRGBA(x, y, ramp[(x+y)%len(ramp)])
		}
	}
}
