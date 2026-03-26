package app

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/WKenya/pixgbc/internal/review"
)

func cleanupInterval(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		return 0
	}
	interval := ttl / 4
	if interval < 15*time.Minute {
		interval = 15 * time.Minute
	}
	if interval > time.Hour {
		interval = time.Hour
	}
	return interval
}

func runStoreCleanup(ctx context.Context, store review.Store, now time.Time) error {
	return store.CleanupExpired(ctx, now.UTC())
}

func startStoreCleanupLoop(ctx context.Context, stdout io.Writer, store review.Store, ttl time.Duration) func() {
	interval := cleanupInterval(ttl)
	if interval <= 0 {
		return func() {}
	}
	_, _ = fmt.Fprintf(stdout, "cleanup loop interval=%s\n", interval)

	ticker := time.NewTicker(interval)
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case tick := <-ticker.C:
				if err := runStoreCleanup(ctx, store, tick); err != nil && ctx.Err() == nil {
					_, _ = fmt.Fprintf(stdout, "cleanup error=%v\n", err)
					continue
				}
				_, _ = fmt.Fprintf(stdout, "cleanup sweep at=%s\n", tick.UTC().Format(time.RFC3339))
			}
		}
	}()

	return func() {
		ticker.Stop()
		<-done
	}
}
