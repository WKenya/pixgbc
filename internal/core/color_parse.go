package core

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
)

func ParseHexColor(raw string) (color.NRGBA, error) {
	value := strings.TrimSpace(raw)
	if strings.HasPrefix(value, "#") {
		value = value[1:]
	}
	if len(value) != 6 {
		return color.NRGBA{}, fmt.Errorf("want #RRGGBB")
	}

	parse := func(pair string) (uint8, error) {
		channel, err := strconv.ParseUint(pair, 16, 8)
		if err != nil {
			return 0, err
		}
		return uint8(channel), nil
	}

	r, err := parse(value[0:2])
	if err != nil {
		return color.NRGBA{}, err
	}
	g, err := parse(value[2:4])
	if err != nil {
		return color.NRGBA{}, err
	}
	b, err := parse(value[4:6])
	if err != nil {
		return color.NRGBA{}, err
	}

	return color.NRGBA{R: r, G: g, B: b, A: 0xFF}, nil
}
