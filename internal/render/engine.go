package render

import (
	"context"

	"github.com/WKenya/pixgbc/internal/core"
)

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Run(ctx context.Context, src core.Source, cfg core.Config) (*core.Result, error) {
	switch cfg.Mode {
	case core.ModeRelaxed:
		return RunRelaxed(ctx, src, cfg)
	case core.ModeCGBBG:
		return RunCGBBG(ctx, src, cfg)
	default:
		return nil, core.ErrUnknownMode
	}
}
