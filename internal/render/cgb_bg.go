package render

import (
	"context"
	"fmt"

	"github.com/WKenya/pixgbc/internal/core"
)

func RunCGBBG(_ context.Context, _ core.Source, _ core.Config) (*core.Result, error) {
	return nil, fmt.Errorf("%w: cgb-bg renderer", core.ErrNotImplemented)
}
