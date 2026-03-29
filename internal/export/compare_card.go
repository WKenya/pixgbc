package export

import (
	"bytes"
	"image"
	"image/color"
	"image/png"

	"github.com/WKenya/pixgbc/internal/core"
)

const (
	compareLabelHeight = 24
	compareGap         = 16
)

var (
	compareBackground = color.NRGBA{R: 0xF1, G: 0xE8, B: 0xD5, A: 0xFF}
	compareLabelBG    = color.NRGBA{R: 0xD8, G: 0xCB, B: 0xAE, A: 0xFF}
	compareLabelText  = color.NRGBA{R: 0x28, G: 0x33, B: 0x1F, A: 0xFF}
)

func CompareCardPNG(source image.Image, result *core.Result) ([]byte, error) {
	img := CompareCardImage(source, result)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func CompareCardImage(source image.Image, result *core.Result) image.Image {
	left := labeledPanel("SOURCE", source)
	rightImage := result.FinalImage
	if result.PreviewImage != nil {
		rightImage = result.PreviewImage
	}
	right := labeledPanel("PIXEL", rightImage)

	width := left.Bounds().Dx() + compareGap + right.Bounds().Dx() + panelPadding*2
	height := max(left.Bounds().Dy(), right.Bounds().Dy()) + panelPadding*2
	card := image.NewNRGBA(image.Rect(0, 0, width, height))
	fillRect(card, card.Bounds(), compareBackground)

	leftBox := image.Rect(panelPadding, panelPadding, panelPadding+left.Bounds().Dx(), panelPadding+left.Bounds().Dy())
	rightBox := image.Rect(leftBox.Max.X+compareGap, panelPadding, leftBox.Max.X+compareGap+right.Bounds().Dx(), panelPadding+right.Bounds().Dy())
	placeCentered(card, left, leftBox)
	placeCentered(card, right, rightBox)
	return card
}

func labeledPanel(label string, img image.Image) *image.NRGBA {
	panel := image.NewNRGBA(image.Rect(0, 0, panelMaxWidth, panelMaxHeight+compareLabelHeight))
	fillRect(panel, panel.Bounds(), panelBackground)

	labelRect := image.Rect(0, 0, panel.Bounds().Dx(), compareLabelHeight)
	fillRect(panel, labelRect, compareLabelBG)
	drawBorder(panel, panel.Bounds(), panelBorder)
	drawTextBlocky(panel, image.Pt(10, 7), label, compareLabelText)

	content := renderPanel(img)
	drawImage(panel, content, image.Pt(0, compareLabelHeight))
	return panel
}

func drawImage(dst *image.NRGBA, src image.Image, at image.Point) {
	bounds := src.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dst.Set(at.X+(x-bounds.Min.X), at.Y+(y-bounds.Min.Y), src.At(x, y))
		}
	}
}

func drawTextBlocky(img *image.NRGBA, origin image.Point, text string, c color.NRGBA) {
	x := origin.X
	for _, r := range text {
		drawGlyph(img, image.Pt(x, origin.Y), r, c)
		x += 12
	}
}

func drawGlyph(img *image.NRGBA, origin image.Point, r rune, c color.NRGBA) {
	pattern, ok := glyphPatterns[r]
	if !ok {
		pattern = glyphPatterns['?']
	}
	for y, row := range pattern {
		for x, ch := range row {
			if ch != '1' {
				continue
			}
			fillRect(img, image.Rect(origin.X+x*2, origin.Y+y*2, origin.X+x*2+2, origin.Y+y*2+2), c)
		}
	}
}

var glyphPatterns = map[rune][]string{
	'A': {"0110", "1001", "1111", "1001", "1001"},
	'C': {"0111", "1000", "1000", "1000", "0111"},
	'E': {"1111", "1000", "1110", "1000", "1111"},
	'I': {"111", "010", "010", "010", "111"},
	'L': {"1000", "1000", "1000", "1000", "1111"},
	'O': {"0110", "1001", "1001", "1001", "0110"},
	'P': {"1110", "1001", "1110", "1000", "1000"},
	'R': {"1110", "1001", "1110", "1010", "1001"},
	'S': {"0111", "1000", "0110", "0001", "1110"},
	'U': {"1001", "1001", "1001", "1001", "0110"},
	'X': {"1001", "1001", "0110", "1001", "1001"},
	'?': {"1110", "0001", "0010", "0000", "0010"},
}
