package app

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"time"

	internalweb "github.com/WKenya/pixgbc/internal/web"
)

func (a *App) runServe(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(a.stderr)

	var addr string
	fs.StringVar(&addr, "addr", "127.0.0.1:8080", "listen address")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	server := &http.Server{
		Addr:    addr,
		Handler: internalweb.NewServer(a.engine(), a.limits),
	}

	_, _ = fmt.Fprintf(a.stdout, "pixgbc listening on http://%s\n", addr)
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_, _ = fmt.Fprintf(a.stderr, "shutdown: %v\n", err)
			return 1
		}
		return 0
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			_, _ = fmt.Fprintf(a.stderr, "serve: %v\n", err)
			return 1
		}
		return 0
	}
}
