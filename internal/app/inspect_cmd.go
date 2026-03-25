package app

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/WKenya/pixgbc/internal/core"
	"github.com/WKenya/pixgbc/internal/ioimg"
	"github.com/WKenya/pixgbc/internal/review"
)

func (a *App) runInspect(args []string) int {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	fs.SetOutput(a.stderr)

	var inputPath string
	fs.StringVar(&inputPath, "input", "", "input image path")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if inputPath == "" {
		_, _ = fmt.Fprintln(a.stderr, "--input required")
		return 2
	}

	file, err := os.Open(inputPath)
	if err != nil {
		_, _ = fmt.Fprintf(a.stderr, "open input: %v\n", err)
		return 1
	}
	defer file.Close()

	decoded, err := ioimg.DecodeImage(file, a.limits)
	if err != nil {
		_, _ = fmt.Fprintf(a.stderr, "decode input: %v\n", err)
		return 1
	}

	out := map[string]any{
		"source":      decoded.Meta,
		"default_cfg": core.DefaultConfig(),
	}

	configHash, err := review.HashConfig(core.DefaultConfig())
	if err == nil {
		out["default_cfg_sha256"] = configHash
	}

	encoder := json.NewEncoder(a.stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(out); err != nil {
		_, _ = fmt.Fprintf(a.stderr, "encode json: %v\n", err)
		return 1
	}

	return 0
}
