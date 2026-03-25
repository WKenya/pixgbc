package app

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/WKenya/pixgbc/internal/core"
	"github.com/WKenya/pixgbc/internal/ioimg"
	"github.com/WKenya/pixgbc/internal/source"
)

func (a *App) runConvert(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("convert", flag.ContinueOnError)
	fs.SetOutput(a.stderr)

	var (
		inputPath    string
		outputPath   string
		size         string
		mode         string
		paletteKey   string
		dither       string
		crop         string
		previewOut   string
		emitReview   string
		paletteMode  string
		previewScale int
	)

	fs.StringVar(&inputPath, "input", "", "input image path")
	fs.StringVar(&outputPath, "output", "", "output PNG path")
	fs.StringVar(&size, "size", "160x144", "target size")
	fs.StringVar(&mode, "mode", "relaxed", "render mode")
	fs.StringVar(&paletteKey, "palette", "gbc-olive", "palette preset")
	fs.StringVar(&paletteMode, "palette-mode", "preset", "palette mode: preset|extract")
	fs.StringVar(&dither, "dither", "ordered", "dither mode")
	fs.StringVar(&crop, "crop", "fill", "crop mode")
	fs.StringVar(&previewOut, "preview-out", "", "optional preview PNG path")
	fs.StringVar(&emitReview, "emit-review", "", "review root dir, or temp")
	fs.IntVar(&previewScale, "preview-scale", 6, "preview upscale factor")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if inputPath == "" || outputPath == "" {
		_, _ = fmt.Fprintln(a.stderr, "--input and --output required")
		return 2
	}

	width, height, err := parseSize(size)
	if err != nil {
		_, _ = fmt.Fprintln(a.stderr, err)
		return 2
	}

	inputBytes, err := os.ReadFile(inputPath)
	if err != nil {
		_, _ = fmt.Fprintf(a.stderr, "read input: %v\n", err)
		return 1
	}

	decoded, err := ioimg.DecodeImage(bytes.NewReader(inputBytes), a.limits)
	if err != nil {
		_, _ = fmt.Fprintf(a.stderr, "decode input: %v\n", err)
		return 1
	}

	cfg := core.Config{
		Mode:            core.Mode(mode),
		TargetWidth:     width,
		TargetHeight:    height,
		PaletteStrategy: core.PaletteStrategy(paletteMode),
		PalettePreset:   paletteKey,
		Dither:          core.DitherMode(dither),
		CropMode:        core.CropMode(crop),
		PreviewScale:    previewScale,
	}

	result, err := a.engine().Run(ctx, source.NewSingleImage(decoded.Image, decoded.Meta), cfg)
	if err != nil {
		_, _ = fmt.Fprintf(a.stderr, "render: %v\n", err)
		return 1
	}

	if err := writePNG(outputPath, result.FinalImage); err != nil {
		_, _ = fmt.Fprintf(a.stderr, "write output: %v\n", err)
		return 1
	}

	if previewOut != "" {
		if err := writePNG(previewOut, result.PreviewImage); err != nil {
			_, _ = fmt.Fprintf(a.stderr, "write preview: %v\n", err)
			return 1
		}
	}

	if emitReview != "" {
		rootDir, reviewDir, err := emitReviewBundle(ctx, emitReview, inputBytes, cfg, result)
		if err != nil {
			_, _ = fmt.Fprintf(a.stderr, "emit review: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintf(a.stdout, "review_root\t%s\nreview_dir\t%s\n", rootDir, reviewDir)
	}

	return 0
}

func parseSize(size string) (int, int, error) {
	parts := strings.Split(strings.ToLower(size), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid size %q; want WIDTHxHEIGHT", size)
	}

	var width, height int
	if _, err := fmt.Sscanf(parts[0], "%d", &width); err != nil {
		return 0, 0, fmt.Errorf("invalid width in %q", size)
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &height); err != nil {
		return 0, 0, fmt.Errorf("invalid height in %q", size)
	}

	return width, height, nil
}
