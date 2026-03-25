package preprocess

import (
	"image"
	"math"
)

func ApplyTone(img *image.NRGBA, brightness, contrast, gamma float64) *image.NRGBA {
	if brightness == 0 && contrast == 0 && gamma == 1 {
		out := image.NewNRGBA(img.Bounds())
		copy(out.Pix, img.Pix)
		return out
	}

	out := image.NewNRGBA(img.Bounds())
	copy(out.Pix, img.Pix)

	for i := 0; i < len(out.Pix); i += 4 {
		out.Pix[i+0] = toneChannel(out.Pix[i+0], brightness, contrast, gamma)
		out.Pix[i+1] = toneChannel(out.Pix[i+1], brightness, contrast, gamma)
		out.Pix[i+2] = toneChannel(out.Pix[i+2], brightness, contrast, gamma)
	}

	return out
}

func toneChannel(v uint8, brightness, contrast, gamma float64) uint8 {
	n := float64(v) / 255.0
	n += brightness
	n = ((n - 0.5) * (1 + contrast)) + 0.5
	n = clampFloat64(n, 0, 1)
	n = math.Pow(n, 1/gamma)
	return uint8(clampFloat64(n*255, 0, 255))
}

func clampFloat64(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
