package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
)

func main() {
	root := "samples"
	must(os.MkdirAll(root, 0o755))

	writePNG(filepath.Join(root, "gradient-landscape.png"), gradientLandscape())
	writePNG(filepath.Join(root, "portrait-alpha.png"), portraitAlpha())
	writePNG(filepath.Join(root, "tile-banks.png"), tileBanks())
}

func gradientLandscape() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 320, 240))
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			r := uint8(20 + x*180/img.Bounds().Dx())
			g := uint8(24 + y*190/img.Bounds().Dy())
			b := uint8(34 + (x+y)*120/(img.Bounds().Dx()+img.Bounds().Dy()))
			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: 0xFF})
		}
	}

	drawStripe(img, image.Rect(0, 140, 320, 240), color.NRGBA{R: 212, G: 176, B: 98, A: 0xFF}, 12)
	drawStripe(img, image.Rect(0, 168, 320, 240), color.NRGBA{R: 108, G: 128, B: 70, A: 0xFF}, 18)
	drawCircle(img, 228, 54, 32, color.NRGBA{R: 245, G: 228, B: 173, A: 0xFF})
	return img
}

func portraitAlpha() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 220, 300))
	fill(img, color.NRGBA{R: 0, G: 0, B: 0, A: 0})

	for y := 0; y < img.Bounds().Dy(); y++ {
		alpha := uint8(90 + y*120/img.Bounds().Dy())
		for x := 0; x < img.Bounds().Dx(); x++ {
			if x < 24 || x > 196 || y < 18 || y > 282 {
				continue
			}
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(36 + x*90/img.Bounds().Dx()),
				G: uint8(54 + y*70/img.Bounds().Dy()),
				B: uint8(98 + (x+y)*40/(img.Bounds().Dx()+img.Bounds().Dy())),
				A: alpha,
			})
		}
	}

	drawCircle(img, 110, 128, 62, color.NRGBA{R: 238, G: 194, B: 152, A: 0xF2})
	drawCircle(img, 88, 116, 10, color.NRGBA{R: 30, G: 34, B: 38, A: 0xFF})
	drawCircle(img, 132, 116, 10, color.NRGBA{R: 30, G: 34, B: 38, A: 0xFF})
	drawStripe(img, image.Rect(80, 160, 140, 170), color.NRGBA{R: 140, G: 74, B: 58, A: 0xE8}, 5)
	return img
}

func tileBanks() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 160, 144))
	palettes := [][]color.NRGBA{
		{{R: 18, G: 26, B: 18, A: 0xFF}, {R: 54, G: 76, B: 48, A: 0xFF}, {R: 128, G: 146, B: 78, A: 0xFF}, {R: 226, G: 234, B: 168, A: 0xFF}},
		{{R: 32, G: 24, B: 22, A: 0xFF}, {R: 78, G: 44, B: 34, A: 0xFF}, {R: 146, G: 88, B: 52, A: 0xFF}, {R: 240, G: 196, B: 126, A: 0xFF}},
		{{R: 18, G: 34, B: 50, A: 0xFF}, {R: 46, G: 76, B: 98, A: 0xFF}, {R: 98, G: 138, B: 162, A: 0xFF}, {R: 216, G: 234, B: 244, A: 0xFF}},
		{{R: 34, G: 18, B: 42, A: 0xFF}, {R: 76, G: 36, B: 84, A: 0xFF}, {R: 136, G: 82, B: 150, A: 0xFF}, {R: 230, G: 186, B: 224, A: 0xFF}},
	}

	for ty := 0; ty < 18; ty++ {
		for tx := 0; tx < 20; tx++ {
			paletteSet := palettes[(tx/5+ty/4)%len(palettes)]
			fillTile(img, tx*8, ty*8, paletteSet)
		}
	}
	return img
}

func fill(img *image.NRGBA, c color.NRGBA) {
	draw.Draw(img, img.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
}

func drawStripe(img *image.NRGBA, rect image.Rectangle, c color.NRGBA, step int) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if ((x+y)/step)%2 == 0 {
				img.SetNRGBA(x, y, c)
			}
		}
	}
}

func drawCircle(img *image.NRGBA, cx, cy, radius int, c color.NRGBA) {
	r2 := radius * radius
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			if !image.Pt(x, y).In(img.Bounds()) {
				continue
			}
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r2 {
				img.SetNRGBA(x, y, c)
			}
		}
	}
}

func fillTile(img *image.NRGBA, minX, minY int, palette []color.NRGBA) {
	for y := minY; y < minY+8; y++ {
		for x := minX; x < minX+8; x++ {
			img.SetNRGBA(x, y, palette[(x+y)%len(palette)])
		}
	}
}

func writePNG(path string, img image.Image) {
	file, err := os.Create(path)
	must(err)
	defer file.Close()
	must(png.Encode(file, img))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
