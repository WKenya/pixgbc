package palette

import (
	"fmt"
	"image"
	"image/color"
	"slices"
)

type ExtractOptions struct {
	Count         int
	GuidedPreset  *Preset
	PreserveBlack bool
}

type histogramEntry struct {
	color uint16
	count int
}

func Extract(img image.Image, opts ExtractOptions) ([]color.NRGBA, error) {
	if opts.Count <= 0 {
		return nil, fmt.Errorf("extract count must be positive")
	}

	histogram := map[uint16]int{}
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if c.A == 0 {
				continue
			}
			histogram[ReduceRGB555(c)]++
		}
	}

	if len(histogram) == 0 {
		return nil, fmt.Errorf("image contains no visible pixels")
	}

	if opts.GuidedPreset != nil {
		for _, c := range opts.GuidedPreset.Colors {
			histogram[ReduceRGB555(c)] += 2
		}
	}

	entries := make([]histogramEntry, 0, len(histogram))
	for bucket, count := range histogram {
		entries = append(entries, histogramEntry{color: bucket, count: count})
	}

	slices.SortFunc(entries, func(a, b histogramEntry) int {
		if a.count != b.count {
			return b.count - a.count
		}
		if a.color < b.color {
			return -1
		}
		if a.color > b.color {
			return 1
		}
		return 0
	})

	palette := make([]color.NRGBA, 0, opts.Count)
	if opts.PreserveBlack {
		palette = append(palette, color.NRGBA{A: 0xFF})
	}

	for _, entry := range entries {
		if len(palette) >= opts.Count {
			break
		}
		candidate := ExpandRGB555(entry.color)
		if containsColor(palette, candidate) {
			continue
		}
		palette = append(palette, candidate)
	}

	for len(palette) < opts.Count {
		palette = append(palette, palette[len(palette)-1])
	}

	slices.SortFunc(palette, compareByLuma)
	return palette, nil
}

func containsColor(colors []color.NRGBA, target color.NRGBA) bool {
	for _, c := range colors {
		if c == target {
			return true
		}
	}

	return false
}

func compareByLuma(a, b color.NRGBA) int {
	la := int(a.R)*2126 + int(a.G)*7152 + int(a.B)*722
	lb := int(b.R)*2126 + int(b.G)*7152 + int(b.B)*722
	if la != lb {
		return la - lb
	}
	if a.R != b.R {
		return int(a.R) - int(b.R)
	}
	if a.G != b.G {
		return int(a.G) - int(b.G)
	}
	return int(a.B) - int(b.B)
}
