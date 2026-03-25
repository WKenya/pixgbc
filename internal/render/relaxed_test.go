package render

import (
	"context"
	"image"
	"image/color"
	"testing"

	"github.com/WKenya/pixgbc/internal/core"
	"github.com/WKenya/pixgbc/internal/palette"
	"github.com/WKenya/pixgbc/internal/source"
)

func TestRunRelaxedProducesTargetSizeAndPresetColors(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 0xFF})
	img.SetNRGBA(1, 0, color.NRGBA{R: 100, G: 100, B: 100, A: 0xFF})
	img.SetNRGBA(0, 1, color.NRGBA{R: 200, G: 210, B: 220, A: 0xFF})
	img.SetNRGBA(1, 1, color.NRGBA{R: 240, G: 240, B: 240, A: 0xFF})

	result, err := RunRelaxed(context.Background(), source.NewSingleImage(img, core.SourceMeta{
		Width:      2,
		Height:     2,
		Format:     "png",
		FileSize:   16,
		FrameCount: 1,
	}), core.Config{
		TargetWidth:   2,
		TargetHeight:  2,
		Dither:        core.DitherNone,
		PreviewScale:  1,
		PalettePreset: "dmg-gray",
	})
	if err != nil {
		t.Fatalf("RunRelaxed() error = %v", err)
	}

	if result.FinalImage.Bounds().Dx() != 2 || result.FinalImage.Bounds().Dy() != 2 {
		t.Fatalf("FinalImage size = %dx%d, want 2x2", result.FinalImage.Bounds().Dx(), result.FinalImage.Bounds().Dy())
	}

	allowed := map[color.NRGBA]struct{}{}
	for _, c := range palette.MustGetPreset("dmg-gray").Colors {
		allowed[c] = struct{}{}
	}

	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			got := color.NRGBAModel.Convert(result.FinalImage.At(x, y)).(color.NRGBA)
			if _, ok := allowed[got]; !ok {
				t.Fatalf("pixel (%d,%d) = %#v not in preset", x, y, got)
			}
		}
	}
}
