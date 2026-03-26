package app

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/WKenya/pixgbc/internal/review"
	internalweb "github.com/WKenya/pixgbc/internal/web"
)

func (a *App) runServe(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(a.stderr)

	var (
		addr           string
		listen         string
		token          string
		artifactTTLRaw string
		maxUploadRaw   string
	)
	fs.StringVar(&addr, "addr", "127.0.0.1:8080", "listen address")
	fs.StringVar(&listen, "listen", "", "listen address")
	fs.StringVar(&token, "token", "", "access token for non-localhost binds")
	fs.StringVar(&artifactTTLRaw, "artifact-ttl", "168h", "artifact retention duration")
	fs.StringVar(&maxUploadRaw, "max-upload-bytes", "", "max upload size, e.g. 10MB")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if listen != "" {
		addr = listen
	}

	artifactTTL, err := time.ParseDuration(artifactTTLRaw)
	if err != nil {
		_, _ = fmt.Fprintf(a.stderr, "invalid --artifact-ttl: %v\n", err)
		return 2
	}
	if !isLocalListen(addr) && token == "" {
		_, _ = fmt.Fprintln(a.stderr, "--token required when binding beyond localhost")
		return 2
	}
	limits := a.limits
	if maxUploadRaw != "" {
		maxUploadBytes, err := parseByteSize(maxUploadRaw)
		if err != nil {
			_, _ = fmt.Fprintf(a.stderr, "invalid --max-upload-bytes: %v\n", err)
			return 2
		}
		limits.MaxFileBytes = maxUploadBytes
	}

	store, err := review.NewTempStore("", artifactTTL)
	if err != nil {
		_, _ = fmt.Fprintf(a.stderr, "init review store: %v\n", err)
		return 1
	}
	if err := runStoreCleanup(ctx, store, time.Now()); err != nil {
		_, _ = fmt.Fprintf(a.stderr, "startup cleanup: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(a.stdout, "startup cleanup complete ttl=%s\n", artifactTTL)

	server := &http.Server{
		Addr: addr,
	}
	handler := internalweb.NewServerWithStore(a.engine(), limits, store, internalweb.ServerConfig{
		Token:     token,
		LogOutput: a.stdout,
	})
	server.Handler = handler
	stopCleanup := startStoreCleanupLoop(ctx, a.stderr, store, artifactTTL)
	defer stopCleanup()

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

func isLocalListen(addr string) bool {
	host := addr
	if strings.Contains(addr, ":") {
		if strings.HasPrefix(addr, "[") {
			end := strings.Index(addr, "]")
			if end > 0 {
				host = addr[1:end]
			}
		} else {
			host = strings.Split(addr, ":")[0]
		}
	}
	switch host {
	case "", "127.0.0.1", "localhost", "::1":
		return true
	default:
		return false
	}
}

func parseByteSize(raw string) (int64, error) {
	value := strings.TrimSpace(strings.ToUpper(raw))
	multiplier := int64(1)
	for _, suffix := range []struct {
		name string
		mul  int64
	}{
		{"KB", 1 << 10},
		{"MB", 1 << 20},
		{"GB", 1 << 30},
		{"B", 1},
	} {
		if strings.HasSuffix(value, suffix.name) {
			value = strings.TrimSpace(strings.TrimSuffix(value, suffix.name))
			multiplier = suffix.mul
			break
		}
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}
	if n <= 0 {
		return 0, fmt.Errorf("must be > 0")
	}
	return n * multiplier, nil
}
