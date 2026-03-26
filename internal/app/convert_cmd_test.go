package app

import (
	"bytes"
	"image/color"
	"testing"

	"github.com/WKenya/pixgbc/internal/core"
)

func TestParseConvertOptions(t *testing.T) {
	var stderr bytes.Buffer
	opts, err := parseConvertOptions([]string{
		"samples/input.png",
		"-o", "out.png",
		"--size", "128x112",
		"--mode", "cgb-bg",
		"--palette-mode", "extract",
		"--dither", "none",
		"--crop", "fit",
		"--scale", "4",
		"--alpha", "reserve-color0",
		"--bg", "#112233",
		"--tile-size", "12",
		"--colors-per-tile", "3",
		"--max-palettes", "6",
		"--brightness", "0.1",
		"--contrast", "-0.2",
		"--gamma", "1.25",
		"--debug",
	}, &stderr)
	if err != nil {
		t.Fatalf("parseConvertOptions error = %v", err)
	}

	if opts.InputPath != "samples/input.png" {
		t.Fatalf("InputPath = %q, want samples/input.png", opts.InputPath)
	}
	if opts.OutputPath != "out.png" {
		t.Fatalf("OutputPath = %q, want out.png", opts.OutputPath)
	}
	if opts.Config.Mode != core.ModeCGBBG {
		t.Fatalf("Mode = %q, want %q", opts.Config.Mode, core.ModeCGBBG)
	}
	if opts.Config.PaletteStrategy != core.PaletteExtract {
		t.Fatalf("PaletteStrategy = %q, want %q", opts.Config.PaletteStrategy, core.PaletteExtract)
	}
	if opts.Config.TargetWidth != 128 || opts.Config.TargetHeight != 112 {
		t.Fatalf("size = %dx%d, want 128x112", opts.Config.TargetWidth, opts.Config.TargetHeight)
	}
	if opts.Config.PreviewScale != 4 {
		t.Fatalf("PreviewScale = %d, want 4", opts.Config.PreviewScale)
	}
	if opts.Config.AlphaMode != core.AlphaReserve {
		t.Fatalf("AlphaMode = %q, want %q", opts.Config.AlphaMode, core.AlphaReserve)
	}
	if opts.Config.TileSize != 12 || opts.Config.ColorsPerTile != 3 || opts.Config.MaxPalettes != 6 {
		t.Fatalf("strict params = %d/%d/%d, want 12/3/6", opts.Config.TileSize, opts.Config.ColorsPerTile, opts.Config.MaxPalettes)
	}
	if opts.Config.Brightness != 0.1 || opts.Config.Contrast != -0.2 || opts.Config.Gamma != 1.25 {
		t.Fatalf("tone params = %v/%v/%v, want 0.1/-0.2/1.25", opts.Config.Brightness, opts.Config.Contrast, opts.Config.Gamma)
	}
	if opts.Config.BackgroundColor != (color.NRGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xFF}) {
		t.Fatalf("BackgroundColor = %#v, want #112233", opts.Config.BackgroundColor)
	}
	if !opts.Config.EmitDebug {
		t.Fatal("EmitDebug = false, want true")
	}
}

func TestParseConvertOptionsInvalidBG(t *testing.T) {
	var stderr bytes.Buffer
	if _, err := parseConvertOptions([]string{
		"--input", "samples/input.png",
		"--output", "out.png",
		"--bg", "#12",
	}, &stderr); err == nil {
		t.Fatal("parseConvertOptions error = nil, want error")
	}
}
