package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/WKenya/pixgbc/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	code := app.New(os.Stdout, os.Stderr).Run(ctx, os.Args[1:])
	os.Exit(code)
}
