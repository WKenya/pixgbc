package app

import (
	"encoding/hex"
	"flag"
	"fmt"
	"image/color"
	"strings"

	"github.com/WKenya/pixgbc/internal/palette"
)

func (a *App) runPalette(args []string) int {
	fs := flag.NewFlagSet("palette", flag.ContinueOnError)
	fs.SetOutput(a.stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	subcommand := "list"
	if fs.NArg() > 0 {
		subcommand = fs.Arg(0)
	}

	switch subcommand {
	case "list":
		for _, preset := range palette.AllPresets() {
			_, _ = fmt.Fprintf(a.stdout, "%s\t%s\t%s\t%s\n", preset.Key, preset.DisplayName, preset.Description, formatColors(preset.Colors))
		}
		return 0
	default:
		_, _ = fmt.Fprintf(a.stderr, "unknown palette subcommand %q\n", subcommand)
		return 1
	}
}

func formatColors(colors []color.NRGBA) string {
	values := make([]string, 0, len(colors))
	for _, c := range colors {
		values = append(values, "#"+hex.EncodeToString([]byte{c.R, c.G, c.B}))
	}
	return strings.Join(values, ",")
}
