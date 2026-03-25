package render

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"testing"

	"github.com/WKenya/pixgbc/internal/core"
	"github.com/WKenya/pixgbc/internal/export"
	"github.com/WKenya/pixgbc/internal/source"
)

func TestGoldenHashRelaxedPreset(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	fillRect(img, img.Bounds(), []color.NRGBA{
		{R: 0x12, G: 0x22, B: 0x18, A: 0xFF},
		{R: 0x40, G: 0x5C, B: 0x38, A: 0xFF},
		{R: 0x86, G: 0x92, B: 0x48, A: 0xFF},
		{R: 0xD4, G: 0xDE, B: 0x8A, A: 0xFF},
	})

	result, err := RunRelaxed(context.Background(), source.NewSingleImage(img, core.SourceMeta{
		Width:      8,
		Height:     8,
		Format:     "png",
		FileSize:   64,
		FrameCount: 1,
	}), core.Config{
		TargetWidth:   16,
		TargetHeight:  16,
		Dither:        core.DitherOrdered,
		PreviewScale:  1,
		PalettePreset: "gbc-olive",
	})
	if err != nil {
		t.Fatalf("RunRelaxed() error = %v", err)
	}

	assertImageHash(t, result.FinalImage, "621f83d54957e22b82eb2dceb714d1b50e8e301c9e049fd3a71b80ffbfff0476")
}

func TestGoldenHashRelaxedExtract(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 10, 6))
	fillRect(img, img.Bounds(), []color.NRGBA{
		{R: 0x20, G: 0x44, B: 0x66, A: 0xFF},
		{R: 0x88, G: 0xAA, B: 0xCC, A: 0xFF},
		{R: 0xD8, G: 0xC8, B: 0x88, A: 0xFF},
		{R: 0xF8, G: 0xF0, B: 0xD8, A: 0xFF},
	})

	result, err := RunRelaxed(context.Background(), source.NewSingleImage(img, core.SourceMeta{
		Width:      10,
		Height:     6,
		Format:     "png",
		FileSize:   60,
		FrameCount: 1,
	}), core.Config{
		TargetWidth:     20,
		TargetHeight:    12,
		Dither:          core.DitherNone,
		PreviewScale:    1,
		PaletteStrategy: core.PaletteExtract,
		PaletteSize:     4,
	})
	if err != nil {
		t.Fatalf("RunRelaxed() error = %v", err)
	}

	assertImageHash(t, result.FinalImage, "b664e2020b4c8cf1ba43f6a54956a1fefd6383d5506d3389a452646157607bd3")
}

func TestGoldenHashCGBBG(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 24, 16))
	fillRect(img, image.Rect(0, 0, 8, 8), []color.NRGBA{
		{R: 0x10, G: 0x18, B: 0x10, A: 0xFF},
		{R: 0x38, G: 0x48, B: 0x30, A: 0xFF},
		{R: 0x80, G: 0x90, B: 0x48, A: 0xFF},
		{R: 0xD8, G: 0xE8, B: 0xA8, A: 0xFF},
	})
	fillRect(img, image.Rect(8, 0, 16, 8), []color.NRGBA{
		{R: 0x20, G: 0x18, B: 0x18, A: 0xFF},
		{R: 0x58, G: 0x30, B: 0x28, A: 0xFF},
		{R: 0xA0, G: 0x68, B: 0x38, A: 0xFF},
		{R: 0xF0, G: 0xC8, B: 0x90, A: 0xFF},
	})
	fillRect(img, image.Rect(16, 0, 24, 8), []color.NRGBA{
		{R: 0x18, G: 0x28, B: 0x38, A: 0xFF},
		{R: 0x40, G: 0x60, B: 0x78, A: 0xFF},
		{R: 0x78, G: 0x98, B: 0xB0, A: 0xFF},
		{R: 0xD0, G: 0xE8, B: 0xF0, A: 0xFF},
	})
	fillRect(img, image.Rect(0, 8, 8, 16), []color.NRGBA{
		{R: 0x12, G: 0x1A, B: 0x12, A: 0xFF},
		{R: 0x3A, G: 0x4C, B: 0x32, A: 0xFF},
		{R: 0x82, G: 0x92, B: 0x4A, A: 0xFF},
		{R: 0xD8, G: 0xE8, B: 0xA8, A: 0xFF},
	})
	fillRect(img, image.Rect(8, 8, 16, 16), []color.NRGBA{
		{R: 0x22, G: 0x1A, B: 0x18, A: 0xFF},
		{R: 0x5A, G: 0x30, B: 0x28, A: 0xFF},
		{R: 0xA2, G: 0x68, B: 0x38, A: 0xFF},
		{R: 0xF0, G: 0xC8, B: 0x90, A: 0xFF},
	})
	fillRect(img, image.Rect(16, 8, 24, 16), []color.NRGBA{
		{R: 0x1A, G: 0x2A, B: 0x3A, A: 0xFF},
		{R: 0x42, G: 0x62, B: 0x7A, A: 0xFF},
		{R: 0x7A, G: 0x9A, B: 0xB2, A: 0xFF},
		{R: 0xD0, G: 0xE8, B: 0xF0, A: 0xFF},
	})

	result, err := RunCGBBG(context.Background(), source.NewSingleImage(img, core.SourceMeta{
		Width:      24,
		Height:     16,
		Format:     "png",
		FileSize:   384,
		FrameCount: 1,
	}), core.Config{
		Mode:          core.ModeCGBBG,
		TargetWidth:   24,
		TargetHeight:  16,
		TileSize:      8,
		ColorsPerTile: 4,
		MaxPalettes:   3,
		PreviewScale:  1,
		Dither:        core.DitherNone,
	})
	if err != nil {
		t.Fatalf("RunCGBBG() error = %v", err)
	}

	assertImageHash(t, result.FinalImage, "723f3a5e9bb95001c4eeccbb6cc38dceb8fc1b78a5653c75df47b582677cc05e")
}

func assertImageHash(t *testing.T, img image.Image, want string) {
	t.Helper()

	pngBytes, err := export.PNGBytes(img)
	if err != nil {
		t.Fatalf("PNGBytes() error = %v", err)
	}

	sum := sha256.Sum256(pngBytes)
	got := hex.EncodeToString(sum[:])
	if got != want {
		t.Fatalf("hash = %s, want %s", got, want)
	}
}
