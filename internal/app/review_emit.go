package app

import (
	"context"
	"image"
	"os"
	"path/filepath"
	"time"

	"github.com/WKenya/pixgbc/internal/core"
	"github.com/WKenya/pixgbc/internal/export"
	"github.com/WKenya/pixgbc/internal/review"
)

func emitReviewBundle(ctx context.Context, root string, inputBytes []byte, cfg core.Config, result *core.Result) (string, string, error) {
	storeRoot := root
	if root == "temp" || root == "auto" {
		storeRoot = ""
	}

	store, err := review.NewTempStore(storeRoot, 7*24*time.Hour)
	if err != nil {
		return "", "", err
	}

	record, err := review.SaveResult(ctx, store, inputBytes, cfg, result)
	if err != nil {
		return "", "", err
	}

	absRoot, err := filepath.Abs(store.RootDir)
	if err != nil {
		absRoot = store.RootDir
	}

	return absRoot, filepath.Join(absRoot, record.ID), nil
}

func writePNG(path string, img image.Image) error {
	data, err := export.PNGBytes(img)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
