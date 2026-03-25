package source

import (
	"context"
	"fmt"
	"image"

	"github.com/WKenya/pixgbc/internal/core"
)

type SingleImage struct {
	frame core.Frame
	meta  core.SourceMeta
}

func NewSingleImage(img image.Image, meta core.SourceMeta) *SingleImage {
	return &SingleImage{
		frame: core.Frame{Image: img, Index: 0},
		meta:  meta,
	}
}

func (s *SingleImage) FrameCount() int {
	return 1
}

func (s *SingleImage) Frame(_ context.Context, i int) (core.Frame, error) {
	if i != 0 {
		return core.Frame{}, fmt.Errorf("frame %d out of range", i)
	}

	return s.frame, nil
}

func (s *SingleImage) Meta() core.SourceMeta {
	return s.meta
}
