package export

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/WKenya/pixgbc/internal/core"
)

func TestCompareCardPNGEncodesCompositeImage(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 48, 32))
	final := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	preview := image.NewNRGBA(image.Rect(0, 0, 96, 96))
	fillImage(source, color.NRGBA{R: 0x30, G: 0x60, B: 0x90, A: 0xFF})
	fillImage(final, color.NRGBA{R: 0x90, G: 0x80, B: 0x30, A: 0xFF})
	fillImage(preview, color.NRGBA{R: 0xC0, G: 0xB0, B: 0x60, A: 0xFF})

	pngBytes, err := CompareCardPNG(source, &core.Result{
		FinalImage:   final,
		PreviewImage: preview,
	})
	if err != nil {
		t.Fatalf("CompareCardPNG() error = %v", err)
	}

	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("png.Decode() error = %v", err)
	}

	if img.Bounds().Dx() <= panelMaxWidth*2 {
		t.Fatalf("compare width = %d, want > %d", img.Bounds().Dx(), panelMaxWidth*2)
	}
	if img.Bounds().Dy() <= panelMaxHeight {
		t.Fatalf("compare height = %d, want > %d", img.Bounds().Dy(), panelMaxHeight)
	}
}
