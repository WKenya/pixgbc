//go:build js && wasm

package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"syscall/js"

	"github.com/WKenya/pixgbc/internal/core"
	"github.com/WKenya/pixgbc/internal/export"
	"github.com/WKenya/pixgbc/internal/palette"
	"github.com/WKenya/pixgbc/internal/render"
	"github.com/WKenya/pixgbc/internal/source"
)

const (
	maxBrowserSourceWidth  = 4096
	maxBrowserSourceHeight = 4096
	maxBrowserSourcePixels = 16 << 20
)

var funcs []js.Func

func main() {
	funcs = []js.Func{
		js.FuncOf(renderJS),
		js.FuncOf(palettesJS),
	}
	js.Global().Set("pixgbcRender", funcs[0])
	js.Global().Set("pixgbcPalettes", funcs[1])
	select {}
}

func renderJS(_ js.Value, args []js.Value) any {
	if len(args) != 1 {
		return errorObject(fmt.Errorf("pixgbcRender expects one payload object"))
	}

	out, err := renderPayload(args[0])
	if err != nil {
		return errorObject(err)
	}
	return out
}

func palettesJS(_ js.Value, _ []js.Value) any {
	presets := palette.AllPresets()
	out := js.Global().Get("Array").New(len(presets))
	for i, preset := range presets {
		item := js.Global().Get("Object").New()
		item.Set("key", preset.Key)
		item.Set("display_name", preset.DisplayName)
		item.Set("description", preset.Description)
		colors := js.Global().Get("Array").New(len(preset.Colors))
		for j, c := range preset.Colors {
			colors.SetIndex(j, colorHex(c))
		}
		item.Set("colors", colors)
		out.SetIndex(i, item)
	}
	return out
}

func renderPayload(payload js.Value) (js.Value, error) {
	width := intProp(payload, "source_width", 0)
	height := intProp(payload, "source_height", 0)
	if width <= 0 || height <= 0 {
		return js.Value{}, fmt.Errorf("source dimensions must be positive")
	}
	if width > maxBrowserSourceWidth || height > maxBrowserSourceHeight {
		return js.Value{}, fmt.Errorf("source dimensions %dx%d exceed browser limit %dx%d", width, height, maxBrowserSourceWidth, maxBrowserSourceHeight)
	}
	if int64(width)*int64(height) > maxBrowserSourcePixels {
		return js.Value{}, fmt.Errorf("source pixels exceed browser limit %d", maxBrowserSourcePixels)
	}

	rgba := payload.Get("rgba")
	if rgba.IsUndefined() || rgba.IsNull() {
		return js.Value{}, fmt.Errorf("missing rgba byte array")
	}
	expected := width * height * 4
	if rgba.Get("byteLength").Int() != expected {
		return js.Value{}, fmt.Errorf("rgba length %d does not match %dx%d", rgba.Get("byteLength").Int(), width, height)
	}

	pix := make([]byte, expected)
	if copied := js.CopyBytesToGo(pix, rgba); copied != expected {
		return js.Value{}, fmt.Errorf("copied %d rgba bytes, want %d", copied, expected)
	}

	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	copy(img.Pix, pix)

	cfg, err := configFromJS(payload)
	if err != nil {
		return js.Value{}, err
	}

	meta := core.SourceMeta{
		Width:      width,
		Height:     height,
		HasAlpha:   hasAlpha(img),
		Format:     "browser-rgba",
		FileSize:   int64(len(pix)),
		FrameCount: 1,
	}
	result, err := render.NewEngine().Run(context.Background(), source.NewSingleImage(img, meta), cfg)
	if err != nil {
		return js.Value{}, err
	}

	finalPNG, err := export.PNGBytes(result.FinalImage)
	if err != nil {
		return js.Value{}, fmt.Errorf("encode final png: %w", err)
	}
	previewPNG, err := export.PNGBytes(result.PreviewImage)
	if err != nil {
		return js.Value{}, fmt.Errorf("encode preview png: %w", err)
	}
	comparePNG, err := export.CompareCardPNG(img, result)
	if err != nil {
		return js.Value{}, fmt.Errorf("encode compare png: %w", err)
	}

	out := js.Global().Get("Object").New()
	out.Set("final_png", jsBytes(finalPNG))
	out.Set("preview_png", jsBytes(previewPNG))
	out.Set("compare_png", jsBytes(comparePNG))
	out.Set("mode", string(cfg.Mode))
	out.Set("width", cfg.TargetWidth)
	out.Set("height", cfg.TargetHeight)

	if cfg.EmitDebug {
		debugPNG, err := export.DebugSheetPNG(img, result)
		if err != nil {
			return js.Value{}, fmt.Errorf("encode debug png: %w", err)
		}
		out.Set("debug_png", jsBytes(debugPNG))
	}

	return out, nil
}

func configFromJS(payload js.Value) (core.Config, error) {
	defaults := core.DefaultConfig()
	cfg := core.Config{
		Mode:            core.Mode(stringProp(payload, "mode", string(defaults.Mode))),
		PaletteStrategy: core.PaletteStrategy(stringProp(payload, "palette_mode", string(defaults.PaletteStrategy))),
		PalettePreset:   stringProp(payload, "palette", defaults.PalettePreset),
		Dither:          core.DitherMode(stringProp(payload, "dither", string(defaults.Dither))),
		CropMode:        core.CropMode(stringProp(payload, "crop", string(defaults.CropMode))),
		AlphaMode:       core.AlphaMode(stringProp(payload, "alpha_mode", string(defaults.AlphaMode))),
		TargetWidth:     intProp(payload, "width", defaults.TargetWidth),
		TargetHeight:    intProp(payload, "height", defaults.TargetHeight),
		Brightness:      floatProp(payload, "brightness", defaults.Brightness),
		Contrast:        floatProp(payload, "contrast", defaults.Contrast),
		Gamma:           floatProp(payload, "gamma", defaults.Gamma),
		PreviewScale:    intProp(payload, "preview_scale", defaults.PreviewScale),
		TileSize:        intProp(payload, "tile_size", defaults.TileSize),
		ColorsPerTile:   intProp(payload, "colors_per_tile", defaults.ColorsPerTile),
		MaxPalettes:     intProp(payload, "max_palettes", defaults.MaxPalettes),
		EmitDebug:       boolProp(payload, "debug", false),
	}
	bg, err := core.ParseHexColor(stringProp(payload, "bg_color", colorHex(defaults.BackgroundColor)))
	if err != nil {
		return core.Config{}, fmt.Errorf("invalid bg_color: %w", err)
	}
	cfg.BackgroundColor = bg
	return cfg, nil
}

func jsBytes(data []byte) js.Value {
	out := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(out, data)
	return out
}

func errorObject(err error) js.Value {
	out := js.Global().Get("Object").New()
	out.Set("error", err.Error())
	return out
}

func stringProp(obj js.Value, name string, fallback string) string {
	value := obj.Get(name)
	if value.IsUndefined() || value.IsNull() || value.String() == "" {
		return fallback
	}
	return value.String()
}

func intProp(obj js.Value, name string, fallback int) int {
	value := obj.Get(name)
	if value.IsUndefined() || value.IsNull() {
		return fallback
	}
	return value.Int()
}

func floatProp(obj js.Value, name string, fallback float64) float64 {
	value := obj.Get(name)
	if value.IsUndefined() || value.IsNull() {
		return fallback
	}
	return value.Float()
}

func boolProp(obj js.Value, name string, fallback bool) bool {
	value := obj.Get(name)
	if value.IsUndefined() || value.IsNull() {
		return fallback
	}
	return value.Bool()
}

func hasAlpha(img *image.NRGBA) bool {
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] != 0xFF {
			return true
		}
	}
	return false
}

func colorHex(c color.NRGBA) string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}
