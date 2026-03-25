package ioimg

import (
	"image"
	"image/png"
	"io"
)

func EncodePNG(w io.Writer, img image.Image) error {
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	return encoder.Encode(w, img)
}
