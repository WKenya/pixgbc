package palette

import (
	"image/color"
	"reflect"
	"testing"
)

func TestClusterTilePalettesRespectsMaxBanksDeterministically(t *testing.T) {
	tilePalettes := [][]color.NRGBA{
		{
			{R: 0x10, G: 0x10, B: 0x10, A: 0xFF},
			{R: 0x30, G: 0x30, B: 0x30, A: 0xFF},
			{R: 0x70, G: 0x70, B: 0x70, A: 0xFF},
			{R: 0xE0, G: 0xE0, B: 0xE0, A: 0xFF},
		},
		{
			{R: 0x18, G: 0x18, B: 0x18, A: 0xFF},
			{R: 0x38, G: 0x38, B: 0x38, A: 0xFF},
			{R: 0x78, G: 0x78, B: 0x78, A: 0xFF},
			{R: 0xE8, G: 0xE8, B: 0xE8, A: 0xFF},
		},
		{
			{R: 0x20, G: 0x40, B: 0x10, A: 0xFF},
			{R: 0x50, G: 0x80, B: 0x20, A: 0xFF},
			{R: 0x90, G: 0xB0, B: 0x50, A: 0xFF},
			{R: 0xE0, G: 0xF0, B: 0xA0, A: 0xFF},
		},
	}

	first := ClusterTilePalettes(tilePalettes, 2, 4)
	second := ClusterTilePalettes(tilePalettes, 2, 4)

	if len(first) != 2 {
		t.Fatalf("len(first) = %d, want 2", len(first))
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("ClusterTilePalettes() not deterministic\nfirst=%#v\nsecond=%#v", first, second)
	}

	assignments := AssignTilePalettesToBanks(tilePalettes, first)
	if len(assignments) != 3 {
		t.Fatalf("len(assignments) = %d, want 3", len(assignments))
	}
	if assignments[0] != assignments[1] {
		t.Fatalf("similar palettes should share a bank: %#v", assignments)
	}
}
