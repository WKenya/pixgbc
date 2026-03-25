package app

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/WKenya/pixgbc/internal/ioimg"
)

func (a *App) runInspect(args []string) int {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	fs.SetOutput(a.stderr)

	var (
		inputPath string
		jsonOut   bool
	)
	fs.StringVar(&inputPath, "input", "", "input image path")
	fs.BoolVar(&jsonOut, "json", true, "emit JSON output")
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
	if !jsonOut {
		_, _ = fmt.Fprintln(a.stderr, "only JSON output is supported right now; use --json")
		return 2
	}

	report, err := buildInspectReport(decoded.Image, decoded.Meta)
	if err != nil {
		_, _ = fmt.Fprintf(a.stderr, "analyze input: %v\n", err)
		return 1
	}

	encoder := json.NewEncoder(a.stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		_, _ = fmt.Fprintf(a.stderr, "encode json: %v\n", err)
		return 1
	}

	return 0
}
