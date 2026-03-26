package palette

import (
	"image"
	"image/color"
	"testing"

	"github.com/WKenya/pixgbc/internal/core"
)

func BenchmarkQuantizeWholeImageOrdered(b *testing.B) {
	img := benchmarkImage(160, 144)
	preset, ok := GetPreset("gbc-olive")
	if !ok {
		b.Fatal("GetPreset(gbc-olive) = false")
	}

	opts := QuantizeOptions{
		Palette: preset.Colors,
		Dither:  core.DitherOrdered,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := QuantizeWholeImage(img, opts); err != nil {
			b.Fatalf("QuantizeWholeImage() error = %v", err)
		}
	}
}

func BenchmarkClusterTilePalettes(b *testing.B) {
	palettes := benchmarkTilePalettes(20 * 18)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		banks := ClusterTilePalettes(palettes, 8, 4)
		assignments := AssignTilePalettesToBanks(palettes, banks)
		if len(assignments) != len(palettes) {
			b.Fatalf("len(assignments) = %d, want %d", len(assignments), len(palettes))
		}
	}
}

func benchmarkImage(width, height int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8((x*3 + y*5) % 256),
				G: uint8((x*7 + y*11) % 256),
				B: uint8((x*13 + y*17) % 256),
				A: 0xFF,
			})
		}
	}
	return img
}

func benchmarkTilePalettes(count int) [][]color.NRGBA {
	out := make([][]color.NRGBA, 0, count)
	for i := 0; i < count; i++ {
		base := uint8((i * 19) % 256)
		out = append(out, []color.NRGBA{
			{R: base, G: uint8((int(base) + 24) % 256), B: uint8((int(base) + 48) % 256), A: 0xFF},
			{R: uint8((int(base) + 36) % 256), G: uint8((int(base) + 60) % 256), B: uint8((int(base) + 84) % 256), A: 0xFF},
			{R: uint8((int(base) + 72) % 256), G: uint8((int(base) + 96) % 256), B: uint8((int(base) + 120) % 256), A: 0xFF},
			{R: uint8((int(base) + 108) % 256), G: uint8((int(base) + 132) % 256), B: uint8((int(base) + 156) % 256), A: 0xFF},
		})
	}
	return out
}
