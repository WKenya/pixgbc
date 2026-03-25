package core

import (
	"context"
	"image"
	"image/color"
	"time"
)

type Mode string

const (
	ModeRelaxed Mode = "relaxed"
	ModeCGBBG   Mode = "cgb-bg"
)

type PaletteStrategy string

const (
	PalettePreset  PaletteStrategy = "preset"
	PaletteExtract PaletteStrategy = "extract"
)

type DitherMode string

const (
	DitherNone    DitherMode = "none"
	DitherOrdered DitherMode = "ordered"
	DitherFS      DitherMode = "floyd-steinberg"
	DitherAtk     DitherMode = "atkinson"
)

type CropMode string

const (
	CropFit  CropMode = "fit"
	CropFill CropMode = "fill"
)

type AlphaMode string

const (
	AlphaFlatten AlphaMode = "flatten"
	AlphaReserve AlphaMode = "reserve-color0"
)

type Config struct {
	Mode            Mode
	TargetWidth     int
	TargetHeight    int
	TileSize        int
	MaxPalettes     int
	ColorsPerTile   int
	PaletteStrategy PaletteStrategy
	PalettePreset   string
	PaletteSize     int
	Dither          DitherMode
	CropMode        CropMode
	Brightness      float64
	Contrast        float64
	Gamma           float64
	PreviewScale    int
	AlphaMode       AlphaMode
	BackgroundColor color.NRGBA
	EmitDebug       bool
}

type SourceMeta struct {
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	HasAlpha   bool   `json:"has_alpha"`
	Format     string `json:"format"`
	FileSize   int64  `json:"file_size"`
	FrameCount int    `json:"frame_count"`
}

type PaletteBank struct {
	Name   string
	Colors []color.NRGBA
}

type TileAssignment struct {
	X           int `json:"x"`
	Y           int `json:"y"`
	PaletteBank int `json:"palette_bank"`
}

type Result struct {
	FinalImage      image.Image
	PreviewImage    image.Image
	NormalizedImage image.Image
	GlobalPalette   []color.NRGBA
	PaletteBanks    []PaletteBank
	TileAssignments []TileAssignment
	DebugImages     map[string]image.Image
	SourceMeta      SourceMeta
	Metadata        map[string]any
}

type Frame struct {
	Image image.Image
	Delay time.Duration
	Index int
}

type Source interface {
	FrameCount() int
	Frame(ctx context.Context, i int) (Frame, error)
	Meta() SourceMeta
}

type Engine interface {
	Run(ctx context.Context, src Source, cfg Config) (*Result, error)
}
