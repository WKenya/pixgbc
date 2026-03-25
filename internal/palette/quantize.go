package palette

import (
	"fmt"
	"image"
	"image/color"

	"github.com/WKenya/pixgbc/internal/core"
)

type QuantizeOptions struct {
	Palette             []color.NRGBA
	Dither              core.DitherMode
	PreserveTransparent bool
}

var ordered4x4 = [4][4]int{
	{0, 8, 2, 10},
	{12, 4, 14, 6},
	{3, 11, 1, 9},
	{15, 7, 13, 5},
}

func QuantizeWholeImage(img image.Image, opts QuantizeOptions) (image.Image, error) {
	if len(opts.Palette) == 0 {
		return nil, fmt.Errorf("palette required")
	}

	bounds := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			src := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if opts.PreserveTransparent && src.A == 0 {
				transparent := opts.Palette[0]
				transparent.A = 0
				out.SetNRGBA(x-bounds.Min.X, y-bounds.Min.Y, transparent)
				continue
			}

			mapped := nearestColor(applyDither(src, x, y, opts.Dither), opts.Palette)
			out.SetNRGBA(x-bounds.Min.X, y-bounds.Min.Y, mapped)
		}
	}

	return out, nil
}

func QuantizeTile(img image.Image, palette []color.NRGBA, dither core.DitherMode) (image.Image, error) {
	return QuantizeWholeImage(img, QuantizeOptions{
		Palette: palette,
		Dither:  dither,
	})
}

func nearestColor(src color.NRGBA, palette []color.NRGBA) color.NRGBA {
	best := palette[0]
	bestDistance := SquaredDistance(src, best)
	for i := 1; i < len(palette); i++ {
		distance := SquaredDistance(src, palette[i])
		if distance < bestDistance {
			best = palette[i]
			bestDistance = distance
		}
	}

	return best
}

func applyDither(src color.NRGBA, x, y int, mode core.DitherMode) color.NRGBA {
	if mode != core.DitherOrdered {
		return src
	}

	threshold := ordered4x4[y%4][x%4] - 8
	adjust := threshold * 6

	return color.NRGBA{
		R: clampUint8(int(src.R) + adjust),
		G: clampUint8(int(src.G) + adjust),
		B: clampUint8(int(src.B) + adjust),
		A: src.A,
	}
}

func clampUint8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
