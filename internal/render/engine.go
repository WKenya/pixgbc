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
	normalized, err := core.NormalizeConfig(cfg)
	if err != nil {
		return nil, err
	}

	switch normalized.Mode {
	case core.ModeRelaxed:
		return RunRelaxed(ctx, src, normalized)
	case core.ModeCGBBG:
		return RunCGBBG(ctx, src, normalized)
	default:
		return nil, core.ErrUnknownMode
	}
}
