package app

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/WKenya/pixgbc/internal/review"
)

type cleanupStore struct {
	mu    sync.Mutex
	calls []time.Time
}

func (s *cleanupStore) Save(context.Context, review.ReviewRecord, map[string][]byte) error {
	return nil
}
func (s *cleanupStore) Get(context.Context, string) (review.ReviewRecord, error) {
	return review.ReviewRecord{}, nil
}
func (s *cleanupStore) OpenArtifact(context.Context, string, string) (io.ReadSeekCloser, error) {
	return nil, nil
}
func (s *cleanupStore) Delete(context.Context, string) error { return nil }
func (s *cleanupStore) CleanupExpired(_ context.Context, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, now)
	return nil
}

func (s *cleanupStore) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

func TestCleanupInterval(t *testing.T) {
	tests := []struct {
		ttl  time.Duration
		want time.Duration
	}{
		{0, 0},
		{10 * time.Minute, 15 * time.Minute},
		{2 * time.Hour, 30 * time.Minute},
		{24 * time.Hour, time.Hour},
	}

	for _, test := range tests {
		if got := cleanupInterval(test.ttl); got != test.want {
			t.Fatalf("cleanupInterval(%v) = %v, want %v", test.ttl, got, test.want)
		}
	}
}

func TestRunStoreCleanup(t *testing.T) {
	store := &cleanupStore{}
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.FixedZone("custom", -4*60*60))

	if err := runStoreCleanup(context.Background(), store, now); err != nil {
		t.Fatalf("runStoreCleanup() error = %v", err)
	}
	if store.callCount() != 1 {
		t.Fatalf("callCount = %d, want 1", store.callCount())
	}
}
