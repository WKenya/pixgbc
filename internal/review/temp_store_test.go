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
		SchemaVersion: CurrentSchemaVersion,
		ID:            "abc123",
		CreatedAt:     time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC),
		Mode:          "relaxed",
		Config:        core.DefaultConfig(),
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
			SourcePNG:  DefaultSourcePNGName,
			FinalPNG:   DefaultFinalPNGName,
			PreviewPNG: DefaultPreviewPNGName,
			ComparePNG: DefaultComparePNGName,
			MetaJSON:   DefaultMetaJSONName,
		},
		Fingerprints: Fingerprints{
			InputSHA256:  "in",
			ConfigSHA256: "cfg",
			OutputSHA256: "out",
		},
	}

	files := map[string][]byte{
		DefaultSourcePNGName:  []byte("source-bytes"),
		DefaultFinalPNGName:   []byte("final-bytes"),
		DefaultPreviewPNGName: []byte("preview-bytes"),
		DefaultComparePNGName: []byte("compare-bytes"),
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
	if got.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("Get().SchemaVersion = %q, want %q", got.SchemaVersion, CurrentSchemaVersion)
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

func TestTempStoreListNewestFirst(t *testing.T) {
	store, err := NewTempStore(t.TempDir(), time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	for _, record := range []ReviewRecord{
		{ID: "old", CreatedAt: time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC), Artifacts: ArtifactManifest{MetaJSON: DefaultMetaJSONName}},
		{ID: "new", CreatedAt: time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC), Artifacts: ArtifactManifest{MetaJSON: DefaultMetaJSONName}},
		{ID: "mid", CreatedAt: time.Date(2026, 3, 25, 11, 0, 0, 0, time.UTC), Artifacts: ArtifactManifest{MetaJSON: DefaultMetaJSONName}},
	} {
		if err := store.Save(context.Background(), record, map[string][]byte{}); err != nil {
			t.Fatalf("Save(%s) error = %v", record.ID, err)
		}
	}

	records, err := store.List(context.Background(), 2)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len(List()) = %d, want 2", len(records))
	}
	if records[0].ID != "new" || records[1].ID != "mid" {
		t.Fatalf("List() order = %q, %q; want new, mid", records[0].ID, records[1].ID)
	}
}
