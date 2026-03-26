package app

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/WKenya/pixgbc/internal/core"
	"github.com/WKenya/pixgbc/internal/ioimg"
	"github.com/WKenya/pixgbc/internal/source"
)

func (a *App) runConvert(ctx context.Context, args []string) int {
	opts, err := parseConvertOptions(args, a.stderr)
	if err != nil {
		return 2
	}

	inputBytes, err := os.ReadFile(opts.InputPath)
	if err != nil {
		_, _ = fmt.Fprintf(a.stderr, "read input: %v\n", err)
		return 1
	}

	decoded, err := ioimg.DecodeImage(bytes.NewReader(inputBytes), a.limits)
	if err != nil {
		_, _ = fmt.Fprintf(a.stderr, "decode input: %v\n", err)
		return 1
	}

	result, err := a.engine().Run(ctx, source.NewSingleImage(decoded.Image, decoded.Meta), opts.Config)
	if err != nil {
		_, _ = fmt.Fprintf(a.stderr, "render: %v\n", err)
		return 1
	}

	if err := writePNG(opts.OutputPath, result.FinalImage); err != nil {
		_, _ = fmt.Fprintf(a.stderr, "write output: %v\n", err)
		return 1
	}

	if opts.PreviewOut != "" {
		if err := writePNG(opts.PreviewOut, result.PreviewImage); err != nil {
			_, _ = fmt.Fprintf(a.stderr, "write preview: %v\n", err)
			return 1
		}
	}

	if opts.EmitReview != "" {
		rootDir, reviewDir, err := emitReviewBundle(ctx, opts.EmitReview, inputBytes, opts.Config, result)
		if err != nil {
			_, _ = fmt.Fprintf(a.stderr, "emit review: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintf(a.stdout, "review_root\t%s\nreview_dir\t%s\n", rootDir, reviewDir)
	}

	return 0
}

type convertOptions struct {
	InputPath  string
	OutputPath string
	PreviewOut string
	EmitReview string
	Config     core.Config
}

func parseConvertOptions(args []string, stderr io.Writer) (convertOptions, error) {
	fs := flag.NewFlagSet("convert", flag.ContinueOnError)
	fs.SetOutput(stderr)

	defaults := core.DefaultConfig()
	var (
		opts          convertOptions
		size          string
		mode          string
		paletteKey    string
		paletteMode   string
		dither        string
		crop          string
		alphaMode     string
		bg            string
		previewScale  int
		scaleAlias    int
		tileSize      int
		colorsPerTile int
		maxPalettes   int
		brightness    float64
		contrast      float64
		gamma         float64
		emitDebug     bool
	)

	fs.StringVar(&opts.InputPath, "input", "", "input image path")
	fs.StringVar(&opts.OutputPath, "output", "", "output PNG path")
	fs.StringVar(&opts.OutputPath, "o", "", "output PNG path")
	fs.StringVar(&size, "size", fmt.Sprintf("%dx%d", defaults.TargetWidth, defaults.TargetHeight), "target size")
	fs.StringVar(&mode, "mode", string(defaults.Mode), "render mode")
	fs.StringVar(&paletteKey, "palette", defaults.PalettePreset, "palette preset")
	fs.StringVar(&paletteMode, "palette-mode", string(defaults.PaletteStrategy), "palette mode: preset|extract")
	fs.StringVar(&dither, "dither", string(defaults.Dither), "dither mode")
	fs.StringVar(&crop, "crop", string(defaults.CropMode), "crop mode")
	fs.StringVar(&opts.PreviewOut, "preview-out", "", "optional preview PNG path")
	fs.StringVar(&opts.EmitReview, "emit-review", "", "review root dir, or temp")
	fs.IntVar(&previewScale, "preview-scale", defaults.PreviewScale, "preview upscale factor")
	fs.IntVar(&scaleAlias, "scale", defaults.PreviewScale, "preview upscale factor")
	fs.StringVar(&alphaMode, "alpha-mode", string(defaults.AlphaMode), "alpha mode")
	fs.StringVar(&alphaMode, "alpha", string(defaults.AlphaMode), "alpha mode")
	fs.StringVar(&bg, "bg", "", "background color in #RRGGBB")
	fs.IntVar(&tileSize, "tile-size", defaults.TileSize, "tile size for strict mode")
	fs.IntVar(&colorsPerTile, "colors-per-tile", defaults.ColorsPerTile, "color budget per tile")
	fs.IntVar(&maxPalettes, "max-palettes", defaults.MaxPalettes, "max shared palettes for strict mode")
	fs.Float64Var(&brightness, "brightness", 0, "tone brightness adjustment")
	fs.Float64Var(&contrast, "contrast", 0, "tone contrast adjustment")
	fs.Float64Var(&gamma, "gamma", defaults.Gamma, "tone gamma adjustment")
	fs.BoolVar(&emitDebug, "debug", false, "emit debug artifacts when supported")

	parseArgs := args
	if len(parseArgs) > 0 && !strings.HasPrefix(parseArgs[0], "-") {
		opts.InputPath = parseArgs[0]
		parseArgs = parseArgs[1:]
	}

	if err := fs.Parse(parseArgs); err != nil {
		return convertOptions{}, err
	}

	if opts.InputPath == "" && fs.NArg() > 0 {
		opts.InputPath = fs.Arg(0)
	}
	if opts.InputPath == "" || opts.OutputPath == "" {
		_, _ = fmt.Fprintln(stderr, "--input and --output required")
		return convertOptions{}, flag.ErrHelp
	}

	width, height, err := parseSize(size)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return convertOptions{}, err
	}

	backgroundColor := defaults.BackgroundColor
	if strings.TrimSpace(bg) != "" {
		backgroundColor, err = core.ParseHexColor(bg)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "invalid --bg: %v\n", err)
			return convertOptions{}, err
		}
	}

	if scaleAlias != defaults.PreviewScale {
		previewScale = scaleAlias
	}

	opts.Config = core.Config{
		Mode:            core.Mode(mode),
		TargetWidth:     width,
		TargetHeight:    height,
		TileSize:        tileSize,
		MaxPalettes:     maxPalettes,
		ColorsPerTile:   colorsPerTile,
		PaletteStrategy: core.PaletteStrategy(paletteMode),
		PalettePreset:   paletteKey,
		Dither:          core.DitherMode(dither),
		CropMode:        core.CropMode(crop),
		Brightness:      brightness,
		Contrast:        contrast,
		Gamma:           gamma,
		PreviewScale:    previewScale,
		AlphaMode:       core.AlphaMode(alphaMode),
		BackgroundColor: backgroundColor,
		EmitDebug:       emitDebug,
	}

	return opts, nil
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
