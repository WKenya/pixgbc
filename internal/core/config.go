package core

import (
	"fmt"
	"image/color"
)

var defaultBackground = color.NRGBA{R: 0xF4, G: 0xF1, B: 0xE8, A: 0xFF}

func DefaultConfig() Config {
	return Config{
		Mode:            ModeRelaxed,
		TargetWidth:     160,
		TargetHeight:    144,
		TileSize:        8,
		MaxPalettes:     8,
		ColorsPerTile:   4,
		PaletteStrategy: PalettePreset,
		PalettePreset:   "gbc-olive",
		PaletteSize:     4,
		Dither:          DitherOrdered,
		CropMode:        CropFill,
		Gamma:           1,
		PreviewScale:    6,
		AlphaMode:       AlphaFlatten,
		BackgroundColor: defaultBackground,
	}
}

func NormalizeConfig(cfg Config) (Config, error) {
	base := DefaultConfig()

	if cfg.Mode == "" {
		cfg.Mode = base.Mode
	}
	if cfg.TargetWidth == 0 {
		cfg.TargetWidth = base.TargetWidth
	}
	if cfg.TargetHeight == 0 {
		cfg.TargetHeight = base.TargetHeight
	}
	if cfg.TileSize == 0 {
		cfg.TileSize = base.TileSize
	}
	if cfg.MaxPalettes == 0 {
		cfg.MaxPalettes = base.MaxPalettes
	}
	if cfg.ColorsPerTile == 0 {
		cfg.ColorsPerTile = base.ColorsPerTile
	}
	if cfg.PaletteStrategy == "" {
		cfg.PaletteStrategy = base.PaletteStrategy
	}
	if cfg.PalettePreset == "" && cfg.PaletteStrategy == PalettePreset {
		cfg.PalettePreset = base.PalettePreset
	}
	if cfg.PaletteSize == 0 {
		cfg.PaletteSize = base.PaletteSize
	}
	if cfg.Dither == "" {
		cfg.Dither = base.Dither
	}
	if cfg.CropMode == "" {
		cfg.CropMode = base.CropMode
	}
	if cfg.Gamma == 0 {
		cfg.Gamma = base.Gamma
	}
	if cfg.PreviewScale == 0 {
		cfg.PreviewScale = base.PreviewScale
	}
	if cfg.AlphaMode == "" {
		cfg.AlphaMode = base.AlphaMode
	}
	if cfg.BackgroundColor.A == 0 {
		cfg.BackgroundColor = base.BackgroundColor
	}

	return cfg, ValidateConfig(cfg)
}

func ValidateConfig(cfg Config) error {
	switch cfg.Mode {
	case ModeRelaxed, ModeCGBBG:
	default:
		return fmt.Errorf("%w: mode %q", ErrUnknownMode, cfg.Mode)
	}

	if cfg.TargetWidth <= 0 || cfg.TargetHeight <= 0 {
		return fmt.Errorf("%w: target size must be positive", ErrInvalidConfig)
	}
	if cfg.TargetWidth > 4096 || cfg.TargetHeight > 4096 {
		return fmt.Errorf("%w: target size too large", ErrInvalidConfig)
	}
	if cfg.TileSize <= 0 || cfg.MaxPalettes <= 0 || cfg.ColorsPerTile <= 0 {
		return fmt.Errorf("%w: tile and palette settings must be positive", ErrInvalidConfig)
	}

	switch cfg.PaletteStrategy {
	case PalettePreset, PaletteExtract:
	default:
		return fmt.Errorf("%w: palette strategy %q", ErrInvalidConfig, cfg.PaletteStrategy)
	}

	if cfg.PaletteStrategy == PalettePreset && cfg.PalettePreset == "" {
		return fmt.Errorf("%w: palette preset required", ErrInvalidConfig)
	}
	if cfg.PaletteSize < 2 || cfg.PaletteSize > 16 {
		return fmt.Errorf("%w: palette size must be between 2 and 16", ErrInvalidConfig)
	}

	switch cfg.Dither {
	case DitherNone, DitherOrdered, DitherFS, DitherAtk:
	default:
		return fmt.Errorf("%w: dither %q", ErrInvalidConfig, cfg.Dither)
	}

	switch cfg.CropMode {
	case CropFit, CropFill:
	default:
		return fmt.Errorf("%w: crop mode %q", ErrInvalidConfig, cfg.CropMode)
	}

	if cfg.Gamma <= 0 {
		return fmt.Errorf("%w: gamma must be > 0", ErrInvalidConfig)
	}
	if cfg.PreviewScale <= 0 || cfg.PreviewScale > 32 {
		return fmt.Errorf("%w: preview scale must be between 1 and 32", ErrInvalidConfig)
	}

	switch cfg.AlphaMode {
	case AlphaFlatten, AlphaReserve:
	default:
		return fmt.Errorf("%w: alpha mode %q", ErrInvalidConfig, cfg.AlphaMode)
	}

	return nil
}
