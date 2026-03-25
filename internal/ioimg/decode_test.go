package ioimg

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestDecodeImageReportsFormatAndAlpha(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 0xFF, A: 0x80})

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}

	decoded, err := DecodeImage(bytes.NewReader(buf.Bytes()), DefaultLimits())
	if err != nil {
		t.Fatalf("DecodeImage() error = %v", err)
	}

	if decoded.Meta.Format != "png" {
		t.Fatalf("Format = %q, want png", decoded.Meta.Format)
	}
	if !decoded.Meta.HasAlpha {
		t.Fatal("HasAlpha = false, want true")
	}
	if decoded.Meta.Width != 2 || decoded.Meta.Height != 2 {
		t.Fatalf("Size = %dx%d, want 2x2", decoded.Meta.Width, decoded.Meta.Height)
	}
}

func TestDecodeConfigAndValidateRejectsOversize(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}

	_, err := DecodeConfigAndValidate(bytes.NewReader(buf.Bytes()), Limits{
		MaxWidth:     3,
		MaxHeight:    4,
		MaxPixels:    16,
		MaxFileBytes: 1 << 20,
	})
	if err == nil {
		t.Fatal("DecodeConfigAndValidate() error = nil, want error")
	}
}
