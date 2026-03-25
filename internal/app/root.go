package app

import (
	"context"
	"fmt"
	"io"

	"github.com/WKenya/pixgbc/internal/ioimg"
	"github.com/WKenya/pixgbc/internal/render"
)

type App struct {
	stdout io.Writer
	stderr io.Writer
	limits ioimg.Limits
}

func New(stdout, stderr io.Writer) *App {
	return &App{
		stdout: stdout,
		stderr: stderr,
		limits: ioimg.DefaultLimits(),
	}
}

func (a *App) Run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		a.printHelp()
		return 0
	}

	switch args[0] {
	case "convert":
		return a.runConvert(ctx, args[1:])
	case "inspect":
		return a.runInspect(args[1:])
	case "palette":
		return a.runPalette(args[1:])
	case "serve":
		return a.runServe(ctx, args[1:])
	case "--help", "-h", "help":
		a.printHelp()
		return 0
	default:
		_, _ = fmt.Fprintf(a.stderr, "unknown command %q\n\n", args[0])
		a.printHelp()
		return 1
	}
}

func (a *App) engine() *render.Engine {
	return render.NewEngine()
}

func (a *App) printHelp() {
	_, _ = fmt.Fprintln(a.stdout, "pixgbc")
	_, _ = fmt.Fprintln(a.stdout, "")
	_, _ = fmt.Fprintln(a.stdout, "commands:")
	_, _ = fmt.Fprintln(a.stdout, "  convert   convert one image into a GBC-style PNG")
	_, _ = fmt.Fprintln(a.stdout, "  inspect   print input image metadata as JSON")
	_, _ = fmt.Fprintln(a.stdout, "  palette   list built-in palettes")
	_, _ = fmt.Fprintln(a.stdout, "  serve     run the local web preview server")
}
