package app

import (
	"image"
	"image/color"
	"slices"
	"strings"

	"github.com/WKenya/pixgbc/internal/core"
	"github.com/WKenya/pixgbc/internal/palette"
	"github.com/WKenya/pixgbc/internal/preprocess"
	"github.com/WKenya/pixgbc/internal/review"
)

type inspectReport struct {
	Source              core.SourceMeta        `json:"source"`
	DominantColors      []string               `json:"dominant_colors"`
	Recommendations     inspectRecommendations `json:"recommendations"`
	StrictModeAnalysis  strictModeAnalysis     `json:"strict_mode_analysis"`
	DefaultConfig       core.Config            `json:"default_cfg"`
	DefaultConfigSHA256 string                 `json:"default_cfg_sha256,omitempty"`
}

type inspectRecommendations struct {
	Mode          string   `json:"mode"`
	PalettePreset string   `json:"palette_preset"`
	Dither        string   `json:"dither"`
	Reasons       []string `json:"reasons,omitempty"`
}

type strictModeAnalysis struct {
	EstimatedUniqueTilePalettes int     `json:"estimated_unique_tile_palettes"`
	EstimatedPaletteBanks       int     `json:"estimated_palette_banks"`
	MeanTileBankDistance        float64 `json:"mean_tile_bank_distance"`
	Suitable                    bool    `json:"suitable"`
}

func buildInspectReport(img image.Image, meta core.SourceMeta) (inspectReport, error) {
	dominantColors, err := palette.Extract(img, palette.ExtractOptions{Count: 6})
	if err != nil {
		return inspectReport{}, err
	}

	presetKey := recommendPreset(dominantColors)
	defaultCfg := core.DefaultConfig()
	configHash, err := review.HashConfig(defaultCfg)
	if err != nil {
		return inspectReport{}, err
	}

	strictAnalysis, reasons := analyzeStrictSuitability(img, meta)
	mode := string(core.ModeRelaxed)
	if strictAnalysis.Suitable {
		mode = string(core.ModeCGBBG)
	}

	recommendations := inspectRecommendations{
		Mode:          mode,
		PalettePreset: presetKey,
		Dither:        string(palette.MustGetPreset(presetKey).RecommendedDither),
		Reasons:       reasons,
	}

	return inspectReport{
		Source:              meta,
		DominantColors:      colorsToHex(dominantColors),
		Recommendations:     recommendations,
		StrictModeAnalysis:  strictAnalysis,
		DefaultConfig:       defaultCfg,
		DefaultConfigSHA256: configHash,
	}, nil
}

func analyzeStrictSuitability(img image.Image, meta core.SourceMeta) (strictModeAnalysis, []string) {
	working := cloneToNRGBA(img)
	if meta.HasAlpha {
		working = preprocess.Flatten(working, core.DefaultConfig().BackgroundColor)
	}
	normalized := preprocess.ResizeToCanvas(
		working,
		core.DefaultConfig().TargetWidth,
		core.DefaultConfig().TargetHeight,
		core.DefaultConfig().CropMode,
		core.DefaultConfig().BackgroundColor,
	)

	tilePalettes, uniqueCount := inspectTilePalettes(normalized)
	banks := palette.ClusterTilePalettes(tilePalettes, core.DefaultConfig().MaxPalettes, core.DefaultConfig().ColorsPerTile)
	assignments := palette.AssignTilePalettesToBanks(tilePalettes, banks)

	totalDistance := 0
	for i, tilePalette := range tilePalettes {
		totalDistance += palette.PaletteDistance(tilePalette, banks[assignments[i]])
	}

	meanDistance := 0.0
	if len(tilePalettes) > 0 {
		meanDistance = float64(totalDistance) / float64(len(tilePalettes))
	}

	suitable := !meta.HasAlpha && meanDistance <= 12000 && uniqueCount <= 48
	reasons := []string{
		"dominant palette matched to nearest preset",
	}
	if meta.HasAlpha {
		reasons = append(reasons, "alpha present; relaxed mode safer")
	} else if suitable {
		reasons = append(reasons, "tile palette fit stayed within the 8-bank strict budget")
	} else {
		reasons = append(reasons, "tile palette diversity suggests relaxed mode will preserve more detail")
	}

	return strictModeAnalysis{
		EstimatedUniqueTilePalettes: uniqueCount,
		EstimatedPaletteBanks:       len(banks),
		MeanTileBankDistance:        meanDistance,
		Suitable:                    suitable,
	}, reasons
}

func inspectTilePalettes(img *image.NRGBA) ([][]color.NRGBA, int) {
	cfg := core.DefaultConfig()
	tileSize := cfg.TileSize
	colorsPerTile := cfg.ColorsPerTile
	gridWidth := (img.Bounds().Dx() + tileSize - 1) / tileSize
	gridHeight := (img.Bounds().Dy() + tileSize - 1) / tileSize

	palettes := make([][]color.NRGBA, 0, gridWidth*gridHeight)
	unique := map[string]struct{}{}
	for gy := 0; gy < gridHeight; gy++ {
		for gx := 0; gx < gridWidth; gx++ {
			rect := image.Rect(
				img.Bounds().Min.X+gx*tileSize,
				img.Bounds().Min.Y+gy*tileSize,
				minInt(img.Bounds().Min.X+(gx+1)*tileSize, img.Bounds().Max.X),
				minInt(img.Bounds().Min.Y+(gy+1)*tileSize, img.Bounds().Max.Y),
			)
			tilePalette, err := palette.Extract(copyTileToOrigin(img, rect), palette.ExtractOptions{Count: colorsPerTile})
			if err != nil {
				continue
			}
			palettes = append(palettes, tilePalette)
			unique[paletteKeyHex(tilePalette)] = struct{}{}
		}
	}

	return palettes, len(unique)
}

func copyTileToOrigin(img *image.NRGBA, rect image.Rectangle) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			out.SetNRGBA(x-rect.Min.X, y-rect.Min.Y, img.NRGBAAt(x, y))
		}
	}
	return out
}

func cloneToNRGBA(img image.Image) *image.NRGBA {
	bounds := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			out.Set(x-bounds.Min.X, y-bounds.Min.Y, img.At(x, y))
		}
	}
	return out
}

func recommendPreset(colors []color.NRGBA) string {
	avgSaturation, avgWarmth := colorProfile(colors)
	bestKey := ""
	bestDistance := 0
	for _, preset := range palette.AllPresets() {
		distance := palette.PaletteDistance(colors, preset.Colors) + presetBias(preset.Key, avgSaturation, avgWarmth)
		if bestKey == "" || distance < bestDistance || (distance == bestDistance && preset.Key < bestKey) {
			bestKey = preset.Key
			bestDistance = distance
		}
	}
	return bestKey
}

func colorProfile(colors []color.NRGBA) (avgSaturation int, avgWarmth int) {
	if len(colors) == 0 {
		return 0, 0
	}

	totalSaturation := 0
	totalWarmth := 0
	for _, c := range colors {
		maxChannel := maxInt3(int(c.R), int(c.G), int(c.B))
		minChannel := minInt3(int(c.R), int(c.G), int(c.B))
		totalSaturation += maxChannel - minChannel
		totalWarmth += int(c.R) - int(c.B)
	}

	return totalSaturation / len(colors), totalWarmth / len(colors)
}

func presetBias(key string, avgSaturation int, avgWarmth int) int {
	switch key {
	case "dmg-gray":
		if avgSaturation > 22 {
			return 50000
		}
		return -4000
	case "lcd-cool":
		if avgWarmth < -10 {
			return -5000
		}
		return 1500
	case "warm-backlight":
		if avgWarmth > 10 {
			return -5000
		}
		return 1500
	case "gbc-olive":
		if avgSaturation >= 12 && avgWarmth >= -8 && avgWarmth <= 12 {
			return -2500
		}
		return 0
	default:
		return 0
	}
}

func colorsToHex(colors []color.NRGBA) []string {
	out := make([]string, 0, len(colors))
	for _, c := range colors {
		out = append(out, colorHex(c))
	}
	return out
}

func colorHex(c color.NRGBA) string {
	return "#" + hex2(c.R) + hex2(c.G) + hex2(c.B)
}

func hex2(v uint8) string {
	const hexdigits = "0123456789abcdef"
	return string([]byte{hexdigits[v>>4], hexdigits[v&0x0F]})
}

func paletteKeyHex(colors []color.NRGBA) string {
	parts := make([]string, 0, len(colors))
	for _, c := range colors {
		parts = append(parts, colorHex(c))
	}
	slices.Sort(parts)
	return strings.Join(parts, ",")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func minInt3(a, b, c int) int {
	return minInt(minInt(a, b), c)
}

func maxInt3(a, b, c int) int {
	if a >= b && a >= c {
		return a
	}
	if b >= a && b >= c {
		return b
	}
	return c
}
