package palette

import (
	"fmt"
	"image/color"
	"slices"

	"github.com/WKenya/pixgbc/internal/core"
)

type Preset struct {
	Key               string
	DisplayName       string
	Description       string
	Colors            []color.NRGBA
	RecommendedDither core.DitherMode
	BrightnessAdjust  float64
	ContrastAdjust    float64
	GammaAdjust       float64
}

var presets = []Preset{
	{
		Key:               "dmg-pea",
		DisplayName:       "DMG Pea",
		Description:       "Classic green pea soup LCD ramp.",
		Colors:            []color.NRGBA{hex("1a1c12"), hex("3b5d3a"), hex("7f9b48"), hex("cfdc95")},
		RecommendedDither: core.DitherOrdered,
	},
	{
		Key:               "dmg-gray",
		DisplayName:       "DMG Gray",
		Description:       "Neutral monochrome with DMG contrast.",
		Colors:            []color.NRGBA{hex("111111"), hex("555555"), hex("aaaaaa"), hex("f3f3f3")},
		RecommendedDither: core.DitherOrdered,
	},
	{
		Key:               "gbc-olive",
		DisplayName:       "GBC Olive",
		Description:       "Balanced olive LCD palette for default conversions.",
		Colors:            []color.NRGBA{hex("1b2a17"), hex("38573a"), hex("7e8f3d"), hex("d1d47a")},
		RecommendedDither: core.DitherOrdered,
	},
	{
		Key:               "gbc-pocket",
		DisplayName:       "GBC Pocket",
		Description:       "Softer contrast with a warmer mid-tone.",
		Colors:            []color.NRGBA{hex("181818"), hex("46504a"), hex("8f9f8a"), hex("e5ead9")},
		RecommendedDither: core.DitherOrdered,
	},
	{
		Key:               "lcd-cool",
		DisplayName:       "LCD Cool",
		Description:       "Cooler blue-green ramp for modern screenshots.",
		Colors:            []color.NRGBA{hex("102026"), hex("24444f"), hex("5e8d92"), hex("d6eef0")},
		RecommendedDither: core.DitherOrdered,
	},
	{
		Key:               "warm-backlight",
		DisplayName:       "Warm Backlight",
		Description:       "Amber-tinted palette for warmer print-like output.",
		Colors:            []color.NRGBA{hex("20130f"), hex("684030"), hex("be8450"), hex("f1ddb0")},
		RecommendedDither: core.DitherOrdered,
	},
}

func AllPresets() []Preset {
	return slices.Clone(presets)
}

func MustGetPreset(key string) Preset {
	preset, ok := GetPreset(key)
	if !ok {
		panic(fmt.Sprintf("unknown palette preset %q", key))
	}
	return preset
}

func GetPreset(key string) (Preset, bool) {
	for _, preset := range presets {
		if preset.Key == key {
			return preset, true
		}
	}

	return Preset{}, false
}

func hex(value string) color.NRGBA {
	var r, g, b uint8
	_, _ = fmt.Sscanf(value, "%02x%02x%02x", &r, &g, &b)
	return color.NRGBA{R: r, G: g, B: b, A: 0xFF}
}
