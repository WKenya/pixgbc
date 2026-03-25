package palette

import "image/color"

func ReduceRGB555(c color.NRGBA) uint16 {
	r := uint16(c.R>>3) << 10
	g := uint16(c.G>>3) << 5
	b := uint16(c.B >> 3)
	return r | g | b
}

func ExpandRGB555(v uint16) color.NRGBA {
	r := uint8((v>>10)&0x1F) << 3
	g := uint8((v>>5)&0x1F) << 3
	b := uint8(v&0x1F) << 3
	return color.NRGBA{R: r, G: g, B: b, A: 0xFF}
}
