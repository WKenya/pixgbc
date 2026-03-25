package preprocess

import (
	"image"
	"image/color"
)

func Flatten(img *image.NRGBA, bg color.NRGBA) *image.NRGBA {
	bounds := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			src := img.NRGBAAt(x, y)
			out.SetNRGBA(x-bounds.Min.X, y-bounds.Min.Y, compositeNRGBA(src, bg))
		}
	}

	return out
}

func compositeNRGBA(src, bg color.NRGBA) color.NRGBA {
	alpha := float64(src.A) / 255.0
	inv := 1 - alpha

	return color.NRGBA{
		R: uint8(clampFloat64(alpha*float64(src.R)+inv*float64(bg.R), 0, 255)),
		G: uint8(clampFloat64(alpha*float64(src.G)+inv*float64(bg.G), 0, 255)),
		B: uint8(clampFloat64(alpha*float64(src.B)+inv*float64(bg.B), 0, 255)),
		A: 0xFF,
	}
}
