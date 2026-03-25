package review

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/WKenya/pixgbc/internal/core"
)

func TestTempStoreSaveGetAndOpenArtifact(t *testing.T) {
	store, err := NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	record := ReviewRecord{
		ID:        "abc123",
		CreatedAt: time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC),
		Mode:      "relaxed",
		Config:    core.DefaultConfig(),
		Source: core.SourceMeta{
			Width:      10,
			Height:     10,
			Format:     "png",
			FileSize:   42,
			FrameCount: 1,
		},
		OutputWidth:  160,
		OutputHeight: 144,
		Artifacts: ArtifactManifest{
			FinalPNG:   DefaultFinalPNGName,
			PreviewPNG: DefaultPreviewPNGName,
			MetaJSON:   DefaultMetaJSONName,
		},
		Fingerprints: Fingerprints{
			InputSHA256:  "in",
			ConfigSHA256: "cfg",
			OutputSHA256: "out",
		},
	}

	files := map[string][]byte{
		DefaultFinalPNGName:   []byte("final-bytes"),
		DefaultPreviewPNGName: []byte("preview-bytes"),
	}

	if err := store.Save(context.Background(), record, files); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Get(context.Background(), record.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != record.ID {
		t.Fatalf("Get().ID = %q, want %q", got.ID, record.ID)
	}
	if got.Fingerprints.OutputSHA256 != "out" {
		t.Fatalf("Get().Fingerprints.OutputSHA256 = %q, want out", got.Fingerprints.OutputSHA256)
	}

	reader, err := store.OpenArtifact(context.Background(), record.ID, DefaultFinalPNGName)
	if err != nil {
		t.Fatalf("OpenArtifact() error = %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if string(data) != "final-bytes" {
		t.Fatalf("artifact contents = %q, want final-bytes", string(data))
	}

	if _, err := store.OpenArtifact(context.Background(), record.ID, "../nope"); err == nil {
		t.Fatal("OpenArtifact() error = nil, want invalid name error")
	}

	if _, err := os.Stat(filepath.Join(store.RootDir, record.ID, DefaultMetaJSONName)); err != nil {
		t.Fatalf("meta.json missing: %v", err)
	}
}

func TestTempStoreCleanupExpired(t *testing.T) {
	store, err := NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	oldRecord := ReviewRecord{
		ID:        "old",
		CreatedAt: time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC),
		Artifacts: ArtifactManifest{MetaJSON: DefaultMetaJSONName},
	}
	newRecord := ReviewRecord{
		ID:        "new",
		CreatedAt: time.Date(2026, 3, 25, 11, 30, 0, 0, time.UTC),
		Artifacts: ArtifactManifest{MetaJSON: DefaultMetaJSONName},
	}

	if err := store.Save(context.Background(), oldRecord, map[string][]byte{}); err != nil {
		t.Fatalf("Save(old) error = %v", err)
	}
	if err := store.Save(context.Background(), newRecord, map[string][]byte{}); err != nil {
		t.Fatalf("Save(new) error = %v", err)
	}

	if err := store.CleanupExpired(context.Background(), time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("CleanupExpired() error = %v", err)
	}

	if _, err := store.Get(context.Background(), "old"); err == nil {
		t.Fatal("Get(old) error = nil, want review not found")
	}
	if _, err := store.Get(context.Background(), "new"); err != nil {
		t.Fatalf("Get(new) error = %v", err)
	}
}
