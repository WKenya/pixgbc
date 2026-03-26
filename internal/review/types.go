package review

import (
	"context"
	"io"
	"time"

	"github.com/WKenya/pixgbc/internal/core"
)

const (
	CurrentSchemaVersion  = "pixgbc.review/v1"
	DefaultFinalPNGName   = "final.png"
	DefaultPreviewPNGName = "preview.png"
	DefaultDebugPNGName   = "debug.png"
	DefaultMetaJSONName   = "meta.json"
)

type ArtifactManifest struct {
	FinalPNG   string `json:"final_png"`
	PreviewPNG string `json:"preview_png"`
	DebugPNG   string `json:"debug_png,omitempty"`
	MetaJSON   string `json:"meta_json"`
}

type Fingerprints struct {
	InputSHA256  string `json:"input_sha256"`
	ConfigSHA256 string `json:"config_sha256"`
	OutputSHA256 string `json:"output_sha256"`
}

type ReviewRecord struct {
	SchemaVersion   string                `json:"schema_version"`
	ID              string                `json:"id"`
	CreatedAt       time.Time             `json:"created_at"`
	Mode            string                `json:"mode"`
	Config          core.Config           `json:"config"`
	Source          core.SourceMeta       `json:"source"`
	OutputWidth     int                   `json:"output_width"`
	OutputHeight    int                   `json:"output_height"`
	GlobalPalette   []string              `json:"global_palette,omitempty"`
	PaletteBanks    [][]string            `json:"palette_banks,omitempty"`
	TileAssignments []core.TileAssignment `json:"tile_assignments,omitempty"`
	Artifacts       ArtifactManifest      `json:"artifacts"`
	Fingerprints    Fingerprints          `json:"fingerprints"`
	Metadata        map[string]any        `json:"metadata,omitempty"`
}

type Store interface {
	Save(ctx context.Context, record ReviewRecord, files map[string][]byte) error
	Get(ctx context.Context, id string) (ReviewRecord, error)
	OpenArtifact(ctx context.Context, id string, name string) (io.ReadSeekCloser, error)
	Delete(ctx context.Context, id string) error
	CleanupExpired(ctx context.Context, now time.Time) error
}
