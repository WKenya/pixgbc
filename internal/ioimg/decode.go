package ioimg

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"

	"github.com/WKenya/pixgbc/internal/core"
)

type Decoded struct {
	Image image.Image
	Meta  core.SourceMeta
}

func DecodeConfigAndValidate(r io.Reader, limits Limits) (core.SourceMeta, error) {
	data, err := readAllWithinLimit(r, limits.MaxFileBytes)
	if err != nil {
		return core.SourceMeta{}, err
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return core.SourceMeta{}, fmt.Errorf("%w: %v", core.ErrUnsupportedFormat, err)
	}

	meta := core.SourceMeta{
		Width:      cfg.Width,
		Height:     cfg.Height,
		Format:     normalizeFormat(format),
		FileSize:   int64(len(data)),
		FrameCount: 1,
	}

	if err := validateMeta(meta, limits); err != nil {
		return core.SourceMeta{}, err
	}

	return meta, nil
}

func DecodeImage(r io.Reader, limits Limits) (*Decoded, error) {
	data, err := readAllWithinLimit(r, limits.MaxFileBytes)
	if err != nil {
		return nil, err
	}

	meta, err := DecodeConfigAndValidate(bytes.NewReader(data), limits)
	if err != nil {
		return nil, err
	}

	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrUnsupportedFormat, err)
	}

	nrgba := normalizeToNRGBA(img)
	meta.Format = normalizeFormat(format)
	meta.HasAlpha = hasAlpha(nrgba)

	return &Decoded{
		Image: nrgba,
		Meta:  meta,
	}, nil
}

func normalizeToNRGBA(img image.Image) *image.NRGBA {
	if existing, ok := img.(*image.NRGBA); ok {
		out := image.NewNRGBA(existing.Bounds())
		copy(out.Pix, existing.Pix)
		return out
	}

	bounds := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			out.SetNRGBA(x-bounds.Min.X, y-bounds.Min.Y, color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA))
		}
	}

	return out
}

func hasAlpha(img *image.NRGBA) bool {
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] != 0xFF {
			return true
		}
	}

	return false
}

func validateMeta(meta core.SourceMeta, limits Limits) error {
	if meta.Width <= 0 || meta.Height <= 0 {
		return fmt.Errorf("%w: invalid image dimensions", core.ErrUnsupportedFormat)
	}
	if meta.Width > limits.MaxWidth || meta.Height > limits.MaxHeight {
		return fmt.Errorf("%w: %dx%d exceeds %dx%d", core.ErrImageTooLarge, meta.Width, meta.Height, limits.MaxWidth, limits.MaxHeight)
	}
	if int64(meta.Width)*int64(meta.Height) > limits.MaxPixels {
		return fmt.Errorf("%w: %d pixels exceeds limit", core.ErrImageTooLarge, int64(meta.Width)*int64(meta.Height))
	}
	if meta.FileSize > limits.MaxFileBytes {
		return fmt.Errorf("%w: %d bytes exceeds limit", core.ErrImageTooLarge, meta.FileSize)
	}

	return nil
}

func readAllWithinLimit(r io.Reader, maxBytes int64) ([]byte, error) {
	limited := io.LimitReader(r, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("%w: file exceeds %d bytes", core.ErrImageTooLarge, maxBytes)
	}

	return data, nil
}

func normalizeFormat(format string) string {
	switch format {
	case "jpeg":
		return "jpg"
	default:
		return format
	}
}
