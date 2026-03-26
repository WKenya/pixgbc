package core

import (
	"image/color"
	"testing"
)

func TestParseHexColor(t *testing.T) {
	got, err := ParseHexColor("#112233")
	if err != nil {
		t.Fatalf("ParseHexColor error = %v", err)
	}
	want := color.NRGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xFF}
	if got != want {
		t.Fatalf("ParseHexColor = %#v, want %#v", got, want)
	}
}

func TestParseHexColorInvalid(t *testing.T) {
	if _, err := ParseHexColor("#12"); err == nil {
		t.Fatal("ParseHexColor error = nil, want error")
	}
}
