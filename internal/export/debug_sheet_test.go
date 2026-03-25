package export

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/WKenya/pixgbc/internal/core"
)

func TestDebugSheetPNGEncodesCompositeImage(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	normalized := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	final := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	preview := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	heatmap := image.NewNRGBA(image.Rect(0, 0, 8, 8))

	fillImage(source, color.NRGBA{R: 0x20, G: 0x30, B: 0x40, A: 0xFF})
	fillImage(normalized, color.NRGBA{R: 0x60, G: 0x70, B: 0x30, A: 0xFF})
	fillImage(final, color.NRGBA{R: 0xA0, G: 0x90, B: 0x50, A: 0xFF})
	fillImage(preview, color.NRGBA{R: 0xD0, G: 0xC0, B: 0x90, A: 0xFF})
	fillImage(heatmap, color.NRGBA{R: 0xF0, G: 0x40, B: 0x30, A: 0xFF})

	pngBytes, err := DebugSheetPNG(source, &core.Result{
		NormalizedImage: normalized,
		FinalImage:      final,
		PreviewImage:    preview,
		GlobalPalette: []color.NRGBA{
			{R: 0x20, G: 0x30, B: 0x40, A: 0xFF},
			{R: 0x60, G: 0x70, B: 0x30, A: 0xFF},
			{R: 0xA0, G: 0x90, B: 0x50, A: 0xFF},
		},
		DebugImages: map[string]image.Image{
			"tile-bank-heatmap": heatmap,
		},
	})
	if err != nil {
		t.Fatalf("DebugSheetPNG() error = %v", err)
	}

	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("png.Decode() error = %v", err)
	}

	if img.Bounds().Dx() <= panelMaxWidth {
		t.Fatalf("sheet width = %d, want > %d", img.Bounds().Dx(), panelMaxWidth)
	}
	if img.Bounds().Dy() <= panelMaxHeight {
		t.Fatalf("sheet height = %d, want > %d", img.Bounds().Dy(), panelMaxHeight)
	}
}

func fillImage(img *image.NRGBA, c color.NRGBA) {
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}
