package preprocess

import (
	"image"
	"image/color"

	"github.com/WKenya/pixgbc/internal/core"
)

func ResizeToCanvas(img *image.NRGBA, width, height int, cropMode core.CropMode, bg color.NRGBA) *image.NRGBA {
	srcW := img.Bounds().Dx()
	srcH := img.Bounds().Dy()
	if srcW == width && srcH == height {
		out := image.NewNRGBA(image.Rect(0, 0, width, height))
		copy(out.Pix, img.Pix)
		return out
	}

	scaleX := float64(width) / float64(srcW)
	scaleY := float64(height) / float64(srcH)
	scale := scaleY
	if cropMode == core.CropFill {
		if scaleX > scaleY {
			scale = scaleX
		}
	} else if scaleX < scaleY {
		scale = scaleX
	}

	targetW := maxInt(1, int(float64(srcW)*scale+0.5))
	targetH := maxInt(1, int(float64(srcH)*scale+0.5))
	scaled := scaleNearest(img, targetW, targetH)

	out := image.NewNRGBA(image.Rect(0, 0, width, height))
	fillNRGBA(out, bg)

	offsetX := (width - targetW) / 2
	offsetY := (height - targetH) / 2
	copyRect(out, scaled, offsetX, offsetY)
	return out
}

func UpscaleNearest(img image.Image, scale int) *image.NRGBA {
	if scale <= 1 {
		return scaleNearest(img, img.Bounds().Dx(), img.Bounds().Dy())
	}

	return scaleNearest(img, img.Bounds().Dx()*scale, img.Bounds().Dy()*scale)
}

func scaleNearest(img image.Image, width, height int) *image.NRGBA {
	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	out := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		srcY := srcBounds.Min.Y + y*srcH/height
		for x := 0; x < width; x++ {
			srcX := srcBounds.Min.X + x*srcW/width
			out.Set(x, y, img.At(srcX, srcY))
		}
	}

	return out
}

func fillNRGBA(img *image.NRGBA, bg color.NRGBA) {
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			img.SetNRGBA(x, y, bg)
		}
	}
}

func copyRect(dst, src *image.NRGBA, offsetX, offsetY int) {
	for y := 0; y < dst.Bounds().Dy(); y++ {
		srcY := y - offsetY
		if srcY < 0 || srcY >= src.Bounds().Dy() {
			continue
		}
		for x := 0; x < dst.Bounds().Dx(); x++ {
			srcX := x - offsetX
			if srcX < 0 || srcX >= src.Bounds().Dx() {
				continue
			}
			dst.SetNRGBA(x, y, src.NRGBAAt(srcX, srcY))
		}
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
