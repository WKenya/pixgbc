package web

import (
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/WKenya/pixgbc/internal/core"
	"github.com/WKenya/pixgbc/internal/review"
)

func TestRenderReviewPageIncludesPaletteAndDistribution(t *testing.T) {
	page, err := renderReviewPage(review.ReviewRecord{
		ID:           "rnd_123",
		CreatedAt:    time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC),
		Mode:         "cgb-bg",
		OutputWidth:  160,
		OutputHeight: 144,
		Config: core.Config{
			Mode:            core.ModeCGBBG,
			TargetWidth:     160,
			TargetHeight:    144,
			PaletteStrategy: core.PalettePreset,
			PalettePreset:   "gbc-olive",
			Dither:          core.DitherOrdered,
			CropMode:        core.CropFill,
			PreviewScale:    6,
			TileSize:        8,
			ColorsPerTile:   4,
			MaxPalettes:     8,
			AlphaMode:       core.AlphaFlatten,
			BackgroundColor: color.NRGBA{R: 0xF4, G: 0xF1, B: 0xE8, A: 0xFF},
			Gamma:           1,
		},
		Source: core.SourceMeta{
			Width:      320,
			Height:     288,
			Format:     "png",
			FileSize:   4096,
			FrameCount: 1,
			HasAlpha:   true,
		},
		GlobalPalette: []string{"#112233", "#445566"},
		PaletteBanks: [][]string{
			{"#112233", "#445566", "#778899", "#aabbcc"},
			{"#ddeeff", "#ccbbaa", "#998877", "#665544"},
		},
		TileAssignments: []core.TileAssignment{
			{X: 0, Y: 0, PaletteBank: 0},
			{X: 1, Y: 0, PaletteBank: 0},
			{X: 0, Y: 1, PaletteBank: 1},
		},
		Fingerprints: review.Fingerprints{
			InputSHA256:  "input-hash",
			ConfigSHA256: "config-hash",
			OutputSHA256: "output-hash",
		},
		Metadata: map[string]any{
			"tile_grid_width":    2,
			"tile_grid_height":   2,
			"palette_bank_count": 2,
		},
	}, "/api/renders/rnd_123", "/source.png", "/preview.png", "/final.png", "/compare.png", "/debug.png")
	if err != nil {
		t.Fatalf("renderReviewPage() error = %v", err)
	}

	body := string(page)
	for _, snippet := range []string{
		"Global Palette",
		"Palette Banks",
		"Tile Bank Distribution",
		"Scaled Preview (6x",
		"Final Native Output (160x144)",
		"Preview is a 6x nearest-neighbor upscale",
		"Original Source",
		"Compare Card",
		"compare.png",
		"Bank 0",
		"2 tiles",
		"#112233",
		"input-hash",
		"debug.png",
		"tile_grid_width",
		`html[data-debug-ui="off"] .debug-only`,
		`localStorage.getItem(storageKey) === "1"`,
	} {
		if !strings.Contains(body, snippet) {
			t.Fatalf("review page missing %q", snippet)
		}
	}
}
