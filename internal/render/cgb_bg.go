package render

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"slices"

	"github.com/WKenya/pixgbc/internal/core"
	"github.com/WKenya/pixgbc/internal/palette"
	"github.com/WKenya/pixgbc/internal/preprocess"
)

type tileJob struct {
	index   int
	gridX   int
	gridY   int
	rect    image.Rectangle
	image   *image.NRGBA
	palette []color.NRGBA
}

func RunCGBBG(ctx context.Context, src core.Source, cfg core.Config) (*core.Result, error) {
	cfg, err := core.NormalizeConfig(cfg)
	if err != nil {
		return nil, err
	}

	frame, err := src.Frame(ctx, 0)
	if err != nil {
		return nil, err
	}

	var selectedPreset *palette.Preset
	if preset, ok := palette.GetPreset(cfg.PalettePreset); ok {
		cfg = applyPresetTuning(cfg, preset)
		selectedPreset = &preset
	} else if cfg.PaletteStrategy == core.PalettePreset {
		return nil, fmt.Errorf("%w: %s", core.ErrUnknownPalette, cfg.PalettePreset)
	}

	bg := cfg.BackgroundColor
	if cfg.AlphaMode == core.AlphaReserve {
		bg = color.NRGBA{}
	}
	normalized := preprocess.ResizeToCanvas(frame.Image, cfg.TargetWidth, cfg.TargetHeight, cfg.CropMode, bg)
	if cfg.AlphaMode == core.AlphaFlatten {
		normalized = preprocess.Flatten(normalized, cfg.BackgroundColor)
	}
	normalized = preprocess.ApplyTone(normalized, cfg.Brightness, cfg.Contrast, cfg.Gamma)

	tiles := splitIntoTiles(normalized, cfg.TileSize)
	tilePalettes := make([][]color.NRGBA, 0, len(tiles))
	for _, tile := range tiles {
		var tilePalette []color.NRGBA
		switch cfg.PaletteStrategy {
		case core.PalettePreset:
			if selectedPreset == nil {
				return nil, fmt.Errorf("%w: %s", core.ErrUnknownPalette, cfg.PalettePreset)
			}
			tilePalette = constrainTileToPreset(tile.image, selectedPreset.Colors, cfg.ColorsPerTile)
		case core.PaletteExtract:
			tilePalette, err = palette.Extract(tile.image, palette.ExtractOptions{
				Count: cfg.ColorsPerTile,
			})
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("%w: %s", core.ErrInvalidConfig, cfg.PaletteStrategy)
		}
		tilePalettes = append(tilePalettes, tilePalette)
	}

	bankPalettes := palette.ClusterTilePalettes(tilePalettes, cfg.MaxPalettes, cfg.ColorsPerTile)
	if cfg.PaletteStrategy == core.PalettePreset && selectedPreset != nil {
		bankPalettes = snapBankPalettesToPreset(bankPalettes, selectedPreset.Colors, cfg.ColorsPerTile)
	}
	assignments := palette.AssignTilePalettesToBanks(tilePalettes, bankPalettes)

	finalImage := image.NewNRGBA(normalized.Bounds())
	tileAssignments := make([]core.TileAssignment, 0, len(tiles))
	for i, tile := range tiles {
		bankIndex := assignments[i]
		quantized, err := palette.QuantizeTile(tile.image, bankPalettes[bankIndex], cfg.Dither)
		if err != nil {
			return nil, err
		}
		blitTile(finalImage, quantized, tile.rect.Min.X, tile.rect.Min.Y)
		tileAssignments = append(tileAssignments, core.TileAssignment{
			X:           tile.gridX,
			Y:           tile.gridY,
			PaletteBank: bankIndex,
		})
	}

	result := &core.Result{
		FinalImage:      finalImage,
		PreviewImage:    preprocess.UpscaleNearest(finalImage, cfg.PreviewScale),
		NormalizedImage: normalized,
		GlobalPalette:   uniqueBankColors(bankPalettes),
		PaletteBanks:    makePaletteBanks(bankPalettes),
		TileAssignments: tileAssignments,
		SourceMeta:      src.Meta(),
		Metadata: map[string]any{
			"mode":                string(cfg.Mode),
			"palette_strategy":    string(cfg.PaletteStrategy),
			"target_width":        cfg.TargetWidth,
			"target_height":       cfg.TargetHeight,
			"tile_size":           cfg.TileSize,
			"tile_grid_width":     tileGridWidth(normalized.Bounds(), cfg.TileSize),
			"tile_grid_height":    tileGridHeight(normalized.Bounds(), cfg.TileSize),
			"palette_bank_count":  len(bankPalettes),
			"palette_assignments": assignments,
		},
	}

	if cfg.EmitDebug {
		result.DebugImages = map[string]image.Image{
			"tile-bank-heatmap": makeTileBankHeatmap(normalized.Bounds(), cfg.TileSize, tiles, assignments, len(bankPalettes)),
		}
	}

	return result, nil
}

func splitIntoTiles(img *image.NRGBA, tileSize int) []tileJob {
	bounds := img.Bounds()
	gridWidth := tileGridWidth(bounds, tileSize)
	gridHeight := tileGridHeight(bounds, tileSize)

	tiles := make([]tileJob, 0, gridWidth*gridHeight)
	index := 0
	for gy := 0; gy < gridHeight; gy++ {
		for gx := 0; gx < gridWidth; gx++ {
			minX := bounds.Min.X + gx*tileSize
			minY := bounds.Min.Y + gy*tileSize
			maxX := min(minX+tileSize, bounds.Max.X)
			maxY := min(minY+tileSize, bounds.Max.Y)
			rect := image.Rect(minX, minY, maxX, maxY)

			tiles = append(tiles, tileJob{
				index: index,
				gridX: gx,
				gridY: gy,
				rect:  rect,
				image: copyRectToOrigin(img, rect),
			})
			index++
		}
	}
	return tiles
}

func copyRectToOrigin(img *image.NRGBA, rect image.Rectangle) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			out.SetNRGBA(x-rect.Min.X, y-rect.Min.Y, img.NRGBAAt(x, y))
		}
	}
	return out
}

func blitTile(dst *image.NRGBA, src image.Image, dstX, dstY int) {
	bounds := src.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dst.Set(dstX+(x-bounds.Min.X), dstY+(y-bounds.Min.Y), src.At(x, y))
		}
	}
}

func makePaletteBanks(bankPalettes [][]color.NRGBA) []core.PaletteBank {
	banks := make([]core.PaletteBank, 0, len(bankPalettes))
	for i, colors := range bankPalettes {
		banks = append(banks, core.PaletteBank{
			Name:   fmt.Sprintf("bank-%d", i),
			Colors: append([]color.NRGBA(nil), colors...),
		})
	}
	return banks
}

func uniqueBankColors(bankPalettes [][]color.NRGBA) []color.NRGBA {
	seen := map[color.NRGBA]struct{}{}
	out := make([]color.NRGBA, 0)
	for _, paletteColors := range bankPalettes {
		for _, c := range paletteColors {
			if _, ok := seen[c]; ok {
				continue
			}
			seen[c] = struct{}{}
			out = append(out, c)
		}
	}
	return out
}

func tileGridWidth(bounds image.Rectangle, tileSize int) int {
	return (bounds.Dx() + tileSize - 1) / tileSize
}

func tileGridHeight(bounds image.Rectangle, tileSize int) int {
	return (bounds.Dy() + tileSize - 1) / tileSize
}

func makeTileBankHeatmap(bounds image.Rectangle, tileSize int, tiles []tileJob, assignments []int, bankCount int) *image.NRGBA {
	out := image.NewNRGBA(bounds)
	heatmapColors := []color.NRGBA{
		{R: 0xD7, G: 0x30, B: 0x27, A: 0xFF},
		{R: 0xFC, G: 0x8D, B: 0x59, A: 0xFF},
		{R: 0xFE, G: 0xE0, B: 0x8B, A: 0xFF},
		{R: 0xD9, G: 0xEF, B: 0x8B, A: 0xFF},
		{R: 0x91, G: 0xCF, B: 0x60, A: 0xFF},
		{R: 0x1A, G: 0x98, B: 0x50, A: 0xFF},
		{R: 0x45, G: 0x75, B: 0xB4, A: 0xFF},
		{R: 0x54, G: 0x24, B: 0x78, A: 0xFF},
	}
	if bankCount == 0 {
		return out
	}

	for i, tile := range tiles {
		fill := heatmapColors[assignments[i]%len(heatmapColors)]
		for y := tile.rect.Min.Y; y < tile.rect.Max.Y; y++ {
			for x := tile.rect.Min.X; x < tile.rect.Max.X; x++ {
				out.SetNRGBA(x, y, fill)
			}
		}
	}
	return out
}

func constrainTileToPreset(img image.Image, presetColors []color.NRGBA, colorsPerTile int) []color.NRGBA {
	if len(presetColors) == 0 || colorsPerTile <= 0 {
		return nil
	}

	type presetHit struct {
		color color.NRGBA
		count int
		index int
	}

	hits := make([]presetHit, 0, len(presetColors))
	for i, c := range presetColors {
		hits = append(hits, presetHit{color: c, index: i})
	}

	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			src := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			best := 0
			bestDistance := palette.SquaredDistance(src, presetColors[0])
			for i := 1; i < len(presetColors); i++ {
				distance := palette.SquaredDistance(src, presetColors[i])
				if distance < bestDistance {
					best = i
					bestDistance = distance
				}
			}
			hits[best].count++
		}
	}

	slices.SortFunc(hits, func(a, b presetHit) int {
		if a.count != b.count {
			return b.count - a.count
		}
		return a.index - b.index
	})

	out := make([]color.NRGBA, 0, colorsPerTile)
	for _, hit := range hits {
		if len(out) >= colorsPerTile {
			break
		}
		out = append(out, hit.color)
	}
	for len(out) < colorsPerTile {
		out = append(out, out[len(out)-1])
	}

	return out
}

func snapBankPalettesToPreset(bankPalettes [][]color.NRGBA, presetColors []color.NRGBA, colorsPerTile int) [][]color.NRGBA {
	out := make([][]color.NRGBA, 0, len(bankPalettes))
	for _, bank := range bankPalettes {
		out = append(out, snapPaletteToPreset(bank, presetColors, colorsPerTile))
	}
	return out
}

func snapPaletteToPreset(colors []color.NRGBA, presetColors []color.NRGBA, colorsPerTile int) []color.NRGBA {
	if len(presetColors) == 0 {
		return append([]color.NRGBA(nil), colors...)
	}

	out := make([]color.NRGBA, 0, colorsPerTile)
	used := make([]bool, len(presetColors))
	for _, candidate := range colors {
		best := -1
		bestDistance := 0
		for i, presetColor := range presetColors {
			if used[i] {
				continue
			}
			distance := palette.SquaredDistance(candidate, presetColor)
			if best == -1 || distance < bestDistance {
				best = i
				bestDistance = distance
			}
		}
		if best == -1 {
			break
		}
		used[best] = true
		out = append(out, presetColors[best])
		if len(out) >= colorsPerTile {
			break
		}
	}

	for _, presetColor := range presetColors {
		if len(out) >= colorsPerTile {
			break
		}
		if slices.Contains(out, presetColor) {
			continue
		}
		out = append(out, presetColor)
	}
	for len(out) < colorsPerTile {
		out = append(out, out[len(out)-1])
	}

	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
