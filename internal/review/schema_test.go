package review

import (
	"context"
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/WKenya/pixgbc/internal/core"
)

func TestNewRecordMaterializesStableSchemaAndDefaults(t *testing.T) {
	record, err := NewRecord(
		"rnd_123",
		time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC),
		core.Config{},
		&core.Result{
			FinalImage:   image.NewNRGBA(image.Rect(0, 0, 160, 144)),
			PreviewImage: image.NewNRGBA(image.Rect(0, 0, 320, 288)),
			SourceMeta: core.SourceMeta{
				Width:      320,
				Height:     288,
				Format:     "png",
				FrameCount: 1,
			},
		},
		Fingerprints{
			InputSHA256:  "input",
			ConfigSHA256: "config",
			OutputSHA256: "output",
		},
		ArtifactManifest{},
	)
	if err != nil {
		t.Fatalf("NewRecord() error = %v", err)
	}

	if record.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("SchemaVersion = %q, want %q", record.SchemaVersion, CurrentSchemaVersion)
	}
	if record.Artifacts.SourcePNG != DefaultSourcePNGName || record.Artifacts.FinalPNG != DefaultFinalPNGName || record.Artifacts.PreviewPNG != DefaultPreviewPNGName || record.Artifacts.ComparePNG != DefaultComparePNGName || record.Artifacts.MetaJSON != DefaultMetaJSONName {
		t.Fatalf("Artifacts = %#v, want default names", record.Artifacts)
	}
	if record.Config.PreviewScale != core.DefaultConfig().PreviewScale {
		t.Fatalf("Config.PreviewScale = %d, want %d", record.Config.PreviewScale, core.DefaultConfig().PreviewScale)
	}
}

func TestTempStoreGetBackfillsSchemaVersion(t *testing.T) {
	rootDir := t.TempDir()
	store, err := NewTempStore(rootDir, time.Hour)
	if err != nil {
		t.Fatalf("NewTempStore() error = %v", err)
	}

	recordDir := filepath.Join(rootDir, "legacy")
	if err := os.MkdirAll(recordDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	legacyJSON, err := json.MarshalIndent(ReviewRecord{
		ID:        "legacy",
		CreatedAt: time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC),
		Artifacts: ArtifactManifest{},
	}, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(recordDir, DefaultMetaJSONName), legacyJSON, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := store.Get(context.Background(), "legacy")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("SchemaVersion = %q, want %q", got.SchemaVersion, CurrentSchemaVersion)
	}
	if got.Artifacts.MetaJSON != DefaultMetaJSONName {
		t.Fatalf("Artifacts.MetaJSON = %q, want %q", got.Artifacts.MetaJSON, DefaultMetaJSONName)
	}
	if got.Artifacts.SourcePNG != DefaultSourcePNGName || got.Artifacts.ComparePNG != DefaultComparePNGName {
		t.Fatalf("Artifacts source/compare = %#v, want default names", got.Artifacts)
	}
}
