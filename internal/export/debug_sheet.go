package export

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"slices"

	"github.com/WKenya/pixgbc/internal/core"
)

const (
	panelPadding   = 16
	panelMaxWidth  = 320
	panelMaxHeight = 240
	swatchHeight   = 28
	swatchGap      = 6
)

var (
	sheetBackground = color.NRGBA{R: 0xF4, G: 0xEF, B: 0xE4, A: 0xFF}
	panelBackground = color.NRGBA{R: 0xFF, G: 0xFC, B: 0xF5, A: 0xFF}
	panelBorder     = color.NRGBA{R: 0x28, G: 0x33, B: 0x1F, A: 0xFF}
)

func DebugSheetPNG(source image.Image, result *core.Result) ([]byte, error) {
	img := DebugSheetImage(source, result)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DebugSheetImage(source image.Image, result *core.Result) image.Image {
	panels := []image.Image{
		renderPanel(source),
		renderPanel(result.NormalizedImage),
		renderPanel(result.FinalImage),
		renderPanel(result.PreviewImage),
	}

	if extra := stableDebugImage(result); extra != nil {
		panels = append(panels, renderPanel(extra))
	}

	columns := 2
	rows := (len(panels) + columns - 1) / columns
	palettePanel := paletteSwatchesPanel(result)

	sheetWidth := columns*panelMaxWidth + (columns+1)*panelPadding
	sheetHeight := rows*panelMaxHeight + (rows+1)*panelPadding + palettePanel.Bounds().Dy() + panelPadding

	sheet := image.NewNRGBA(image.Rect(0, 0, sheetWidth, sheetHeight))
	fillRect(sheet, sheet.Bounds(), sheetBackground)

	for i, panel := range panels {
		row := i / columns
		col := i % columns
		x := panelPadding + col*(panelMaxWidth+panelPadding)
		y := panelPadding + row*(panelMaxHeight+panelPadding)
		placeCentered(sheet, panel, image.Rect(x, y, x+panelMaxWidth, y+panelMaxHeight))
	}

	paletteY := panelPadding + rows*(panelMaxHeight+panelPadding)
	placeCentered(sheet, palettePanel, image.Rect(panelPadding, paletteY, sheetWidth-panelPadding, paletteY+palettePanel.Bounds().Dy()))

	return sheet
}

func renderPanel(img image.Image) *image.NRGBA {
	panel := image.NewNRGBA(image.Rect(0, 0, panelMaxWidth, panelMaxHeight))
	fillRect(panel, panel.Bounds(), panelBackground)
	drawBorder(panel, panel.Bounds(), panelBorder)

	if img == nil || img.Bounds().Dx() == 0 || img.Bounds().Dy() == 0 {
		return panel
	}

	inner := image.Rect(8, 8, panelMaxWidth-8, panelMaxHeight-8)
	scaled := scaleToFit(img, inner.Dx(), inner.Dy())
	placeCentered(panel, scaled, inner)
	return panel
}

func paletteSwatchesPanel(result *core.Result) *image.NRGBA {
	rows := bankRows(result)
	if len(rows) == 0 {
		rows = [][]color.NRGBA{{panelBorder}}
	}

	maxCols := 1
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	width := maxCols*(swatchHeight*2) + (maxCols-1)*swatchGap + 24
	height := len(rows)*swatchHeight + (len(rows)-1)*swatchGap + 24
	panel := image.NewNRGBA(image.Rect(0, 0, width, height))
	fillRect(panel, panel.Bounds(), panelBackground)
	drawBorder(panel, panel.Bounds(), panelBorder)

	y := 12
	for _, row := range rows {
		x := 12
		for _, swatch := range row {
			rect := image.Rect(x, y, x+swatchHeight*2, y+swatchHeight)
			fillRect(panel, rect, swatch)
			drawBorder(panel, rect, panelBorder)
			x += swatchHeight*2 + swatchGap
		}
		y += swatchHeight + swatchGap
	}

	return panel
}

func bankRows(result *core.Result) [][]color.NRGBA {
	if len(result.PaletteBanks) > 0 {
		rows := make([][]color.NRGBA, 0, len(result.PaletteBanks))
		for _, bank := range result.PaletteBanks {
			rows = append(rows, append([]color.NRGBA(nil), bank.Colors...))
		}
		return rows
	}
	if len(result.GlobalPalette) > 0 {
		return [][]color.NRGBA{append([]color.NRGBA(nil), result.GlobalPalette...)}
	}
	return nil
}

func scaleToFit(img image.Image, maxWidth, maxHeight int) *image.NRGBA {
	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()
	if srcW <= 0 || srcH <= 0 {
		return image.NewNRGBA(image.Rect(0, 0, 1, 1))
	}

	scaleX := float64(maxWidth) / float64(srcW)
	scaleY := float64(maxHeight) / float64(srcH)
	scale := minFloat(scaleX, scaleY)
	if scale >= 1 {
		scale = float64(max(1, int(scale)))
	}

	dstW := max(1, int(float64(srcW)*scale))
	dstH := max(1, int(float64(srcH)*scale))
	return scaleNearest(img, dstW, dstH)
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

func placeCentered(dst *image.NRGBA, src image.Image, box image.Rectangle) {
	x := box.Min.X + (box.Dx()-src.Bounds().Dx())/2
	y := box.Min.Y + (box.Dy()-src.Bounds().Dy())/2
	draw.Draw(dst, image.Rect(x, y, x+src.Bounds().Dx(), y+src.Bounds().Dy()), src, src.Bounds().Min, draw.Src)
}

func fillRect(img *image.NRGBA, rect image.Rectangle, c color.NRGBA) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}

func drawBorder(img *image.NRGBA, rect image.Rectangle, c color.NRGBA) {
	if rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}
	for x := rect.Min.X; x < rect.Max.X; x++ {
		img.SetNRGBA(x, rect.Min.Y, c)
		img.SetNRGBA(x, rect.Max.Y-1, c)
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		img.SetNRGBA(rect.Min.X, y, c)
		img.SetNRGBA(rect.Max.X-1, y, c)
	}
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func stableDebugImage(result *core.Result) image.Image {
	if result == nil || len(result.DebugImages) == 0 {
		return nil
	}
	if img, ok := result.DebugImages["tile-bank-heatmap"]; ok {
		return img
	}
	keys := make([]string, 0, len(result.DebugImages))
	for key := range result.DebugImages {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return result.DebugImages[keys[0]]
}
