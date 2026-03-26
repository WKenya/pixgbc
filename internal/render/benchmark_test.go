package render

import (
	"context"
	"image"
	"image/color"
	"testing"

	"github.com/WKenya/pixgbc/internal/core"
	"github.com/WKenya/pixgbc/internal/source"
)

func BenchmarkEngineRelaxedPreset(b *testing.B) {
	engine := NewEngine()
	src := benchmarkSource(320, 288, true)
	cfg := core.Config{
		Mode:          core.ModeRelaxed,
		TargetWidth:   160,
		TargetHeight:  144,
		PalettePreset: "gbc-olive",
		Dither:        core.DitherOrdered,
		CropMode:      core.CropFill,
		PreviewScale:  6,
		AlphaMode:     core.AlphaFlatten,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := engine.Run(context.Background(), src, cfg); err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}

func BenchmarkEngineRelaxedExtract(b *testing.B) {
	engine := NewEngine()
	src := benchmarkSource(320, 288, false)
	cfg := core.Config{
		Mode:            core.ModeRelaxed,
		TargetWidth:     160,
		TargetHeight:    144,
		PaletteStrategy: core.PaletteExtract,
		Dither:          core.DitherOrdered,
		CropMode:        core.CropFill,
		PreviewScale:    6,
		AlphaMode:       core.AlphaFlatten,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := engine.Run(context.Background(), src, cfg); err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}

func BenchmarkEngineCGBBG(b *testing.B) {
	engine := NewEngine()
	src := benchmarkSource(320, 288, false)
	cfg := core.Config{
		Mode:          core.ModeCGBBG,
		TargetWidth:   160,
		TargetHeight:  144,
		PalettePreset: "gbc-olive",
		Dither:        core.DitherOrdered,
		CropMode:      core.CropFill,
		PreviewScale:  6,
		AlphaMode:     core.AlphaFlatten,
		TileSize:      8,
		ColorsPerTile: 4,
		MaxPalettes:   8,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := engine.Run(context.Background(), src, cfg); err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}

func BenchmarkEngineCGBBGLargeInput(b *testing.B) {
	engine := NewEngine()
	src := benchmarkSource(2048, 1536, false)
	cfg := core.Config{
		Mode:          core.ModeCGBBG,
		TargetWidth:   160,
		TargetHeight:  144,
		PalettePreset: "gbc-olive",
		Dither:        core.DitherOrdered,
		CropMode:      core.CropFill,
		PreviewScale:  6,
		AlphaMode:     core.AlphaFlatten,
		TileSize:      8,
		ColorsPerTile: 4,
		MaxPalettes:   8,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := engine.Run(context.Background(), src, cfg); err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}

func benchmarkSource(width, height int, withAlpha bool) *source.SingleImage {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			a := uint8(0xFF)
			if withAlpha && (x/24+y/18)%5 == 0 {
				a = uint8(0x60 + ((x + y) % 128))
			}
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8((x*5 + y*3) % 256),
				G: uint8((x*2 + y*7) % 256),
				B: uint8((x*11 + y*13) % 256),
				A: a,
			})
		}
	}

	return source.NewSingleImage(img, core.SourceMeta{
		Width:      width,
		Height:     height,
		HasAlpha:   withAlpha,
		Format:     "png",
		FrameCount: 1,
	})
}
