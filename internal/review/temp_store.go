package review

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/WKenya/pixgbc/internal/core"
)

type TempStore struct {
	RootDir string
	TTL     time.Duration
}

func NewTempStore(rootDir string, ttl time.Duration) (*TempStore, error) {
	if rootDir == "" {
		dir, err := os.MkdirTemp("", "pixgbc-reviews-*")
		if err != nil {
			return nil, err
		}
		rootDir = dir
	}

	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return nil, err
	}

	return &TempStore{RootDir: rootDir, TTL: ttl}, nil
}

func (s *TempStore) Save(ctx context.Context, record ReviewRecord, files map[string][]byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if record.ID == "" {
		return fmt.Errorf("%w: review id required", core.ErrInvalidConfig)
	}

	record = normalizeReviewRecord(record)
	finalDir := filepath.Join(s.RootDir, record.ID)
	if _, err := os.Stat(finalDir); err == nil {
		return fmt.Errorf("review %q already exists", record.ID)
	}

	tmpDir, err := os.MkdirTemp(s.RootDir, "tmp-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	for name, data := range files {
		cleanName, err := cleanArtifactName(name)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(tmpDir, cleanName), data, 0o644); err != nil {
			return err
		}
	}

	metaBytes, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmpDir, record.Artifacts.MetaJSON), metaBytes, 0o644); err != nil {
		return err
	}

	if err := os.Rename(tmpDir, finalDir); err != nil {
		return err
	}

	return nil
}

func (s *TempStore) Get(ctx context.Context, id string) (ReviewRecord, error) {
	if err := ctx.Err(); err != nil {
		return ReviewRecord{}, err
	}

	data, err := os.ReadFile(filepath.Join(s.RootDir, id, DefaultMetaJSONName))
	if errors.Is(err, fs.ErrNotExist) {
		return ReviewRecord{}, core.ErrReviewNotFound
	}
	if err != nil {
		return ReviewRecord{}, err
	}

	var record ReviewRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return ReviewRecord{}, err
	}

	return normalizeReviewRecord(record), nil
}

func (s *TempStore) OpenArtifact(ctx context.Context, id string, name string) (io.ReadSeekCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	cleanName, err := cleanArtifactName(name)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(filepath.Join(s.RootDir, id, cleanName))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, core.ErrReviewNotFound
	}
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (s *TempStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(s.RootDir, id)); err != nil {
		return err
	}
	return nil
}

func (s *TempStore) CleanupExpired(ctx context.Context, now time.Time) error {
	if s.TTL <= 0 {
		return nil
	}

	entries, err := os.ReadDir(s.RootDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), "tmp-") {
			continue
		}

		record, err := s.Get(ctx, entry.Name())
		if errors.Is(err, core.ErrReviewNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		if now.UTC().Sub(record.CreatedAt.UTC()) <= s.TTL {
			continue
		}
		if err := s.Delete(ctx, entry.Name()); err != nil {
			return err
		}
	}

	return nil
}

func cleanArtifactName(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("%w: artifact name required", core.ErrInvalidConfig)
	}

	clean := filepath.Clean(name)
	if clean == "." || clean == ".." || strings.Contains(clean, string(filepath.Separator)) || filepath.IsAbs(clean) {
		return "", fmt.Errorf("%w: invalid artifact name %q", core.ErrInvalidConfig, name)
	}

	return clean, nil
}
