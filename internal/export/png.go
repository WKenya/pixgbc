package export

import (
	"bytes"
	"image"

	"github.com/WKenya/pixgbc/internal/ioimg"
)

func PNGBytes(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := ioimg.EncodePNG(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
