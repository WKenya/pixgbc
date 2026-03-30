package render

import (
	"context"
	"fmt"
	"image"
	"image/color"

	"github.com/WKenya/pixgbc/internal/core"
	"github.com/WKenya/pixgbc/internal/palette"
	"github.com/WKenya/pixgbc/internal/preprocess"
)

func RunRelaxed(ctx context.Context, src core.Source, cfg core.Config) (*core.Result, error) {
	cfg, err := core.NormalizeConfig(cfg)
	if err != nil {
		return nil, err
	}

	core.ReportProgress(ctx, "source", 14, "loading source frame")
	frame, err := src.Frame(ctx, 0)
	if err != nil {
		return nil, err
	}

	preset, hasPreset := palette.GetPreset(cfg.PalettePreset)
	if hasPreset {
		cfg = applyPresetTuning(cfg, preset)
	}

	bg := cfg.BackgroundColor
	if cfg.AlphaMode == core.AlphaReserve {
		bg = color.NRGBA{}
	}
	core.ReportProgress(ctx, "preprocess", 28, "resizing and tone mapping")
	normalized := preprocess.ResizeToCanvas(frame.Image, cfg.TargetWidth, cfg.TargetHeight, cfg.CropMode, bg)
	if cfg.AlphaMode == core.AlphaFlatten {
		normalized = preprocess.Flatten(normalized, cfg.BackgroundColor)
	}
	normalized = preprocess.ApplyTone(normalized, cfg.Brightness, cfg.Contrast, cfg.Gamma)
	core.ReportProgress(ctx, "palette", 54, "resolving palette")
	paletteColors, err := resolvePalette(normalized, cfg)
	if err != nil {
		return nil, err
	}

	core.ReportProgress(ctx, "quantize", 78, "quantizing image")
	finalImage, err := palette.QuantizeWholeImage(normalized, palette.QuantizeOptions{
		Palette:             paletteColors,
		Dither:              cfg.Dither,
		PreserveTransparent: cfg.AlphaMode == core.AlphaReserve,
	})
	if err != nil {
		return nil, err
	}

	core.ReportProgress(ctx, "preview", 92, "building preview")
	preview := preprocess.UpscaleNearest(finalImage, cfg.PreviewScale)

	return &core.Result{
		FinalImage:      finalImage,
		PreviewImage:    preview,
		NormalizedImage: normalized,
		GlobalPalette:   paletteColors,
		SourceMeta:      src.Meta(),
		Metadata: map[string]any{
			"mode":             string(cfg.Mode),
			"palette_strategy": string(cfg.PaletteStrategy),
			"target_width":     cfg.TargetWidth,
			"target_height":    cfg.TargetHeight,
		},
	}, nil
}

func resolvePalette(img image.Image, cfg core.Config) ([]color.NRGBA, error) {
	switch cfg.PaletteStrategy {
	case core.PalettePreset:
		preset, ok := palette.GetPreset(cfg.PalettePreset)
		if !ok {
			return nil, fmt.Errorf("%w: %s", core.ErrUnknownPalette, cfg.PalettePreset)
		}
		return preset.Colors, nil
	case core.PaletteExtract:
		return palette.Extract(img, palette.ExtractOptions{
			Count: cfg.PaletteSize,
		})
	default:
		return nil, fmt.Errorf("%w: %s", core.ErrInvalidConfig, cfg.PaletteStrategy)
	}
}

func applyPresetTuning(cfg core.Config, preset palette.Preset) core.Config {
	if cfg.Dither == "" {
		cfg.Dither = preset.RecommendedDither
	}
	if cfg.Brightness == 0 {
		cfg.Brightness = preset.BrightnessAdjust
	}
	if cfg.Contrast == 0 {
		cfg.Contrast = preset.ContrastAdjust
	}
	if cfg.Gamma == 1 && preset.GammaAdjust != 0 {
		cfg.Gamma = preset.GammaAdjust
	}
	return cfg
}
