# pixgbc API and Interface Specification

## Status
Draft v1

## Engine boundary
The core engine must be usable by CLI tests, HTTP handlers, and future batch/animation workflows without depending on either Cobra or `net/http`.

## Go package responsibilities

### `internal/core`
Owns shared configuration, mode enums, public engine result types, and common error values.

### `internal/source`
Owns source abstractions. In v1, only a single-image source is implemented. The abstraction exists so GIF/video can be added later without rewriting the engine.

### `internal/ioimg`
Owns input decoding, output encoding, size/config validation, and format normalization.

### `internal/preprocess`
Owns crop, aspect handling, resize, tonal conditioning, and alpha preparation.

### `internal/palette`
Owns preset definitions, extraction, quantization distance functions, RGB555 reduction helpers, and palette-bank clustering.

### `internal/render`
Owns the actual rendering modes: `relaxed` and `cgb-bg`.

### `internal/review`
Owns review records, artifact manifests, hashing, and temp-disk storage for locally hosted review artifacts.

### `internal/web`
Owns HTTP DTOs, handlers, HTML views, and static asset serving.

## Core Go types

```go
package core

import (
    "context"
    "image"
    "image/color"
    "time"
)

type Mode string

const (
    ModeRelaxed Mode = "relaxed"
    ModeCGBBG   Mode = "cgb-bg"
)

type PaletteStrategy string

const (
    PalettePreset  PaletteStrategy = "preset"
    PaletteExtract PaletteStrategy = "extract"
)

type DitherMode string

const (
    DitherNone    DitherMode = "none"
    DitherOrdered DitherMode = "ordered"
    DitherFS      DitherMode = "floyd-steinberg"
    DitherAtk     DitherMode = "atkinson"
)

type CropMode string

const (
    CropFit  CropMode = "fit"
    CropFill CropMode = "fill"
)

type AlphaMode string

const (
    AlphaFlatten AlphaMode = "flatten"
    AlphaReserve AlphaMode = "reserve-color0"
)

type Config struct {
    Mode            Mode
    TargetWidth     int
    TargetHeight    int
    TileSize        int
    MaxPalettes     int
    ColorsPerTile   int
    PaletteStrategy PaletteStrategy
    PalettePreset   string
    PaletteSize     int
    Dither          DitherMode
    CropMode        CropMode
    Brightness      float64
    Contrast        float64
    Gamma           float64
    PreviewScale    int
    AlphaMode       AlphaMode
    BackgroundColor color.NRGBA
    EmitDebug       bool
}

type SourceMeta struct {
    Width     int
    Height    int
    HasAlpha  bool
    Format    string
    FileSize  int64
    FrameCount int
}

type PaletteBank struct {
    Name   string
    Colors []color.NRGBA
}

type TileAssignment struct {
    X           int
    Y           int
    PaletteBank int
}

type Result struct {
    FinalImage      image.Image
    PreviewImage    image.Image
    NormalizedImage image.Image
    GlobalPalette   []color.NRGBA
    PaletteBanks    []PaletteBank
    TileAssignments []TileAssignment
    DebugImages     map[string]image.Image
    SourceMeta      SourceMeta
    Metadata        map[string]any
}

type Frame struct {
    Image image.Image
    Delay time.Duration
    Index int
}

type Source interface {
    FrameCount() int
    Frame(ctx context.Context, i int) (Frame, error)
    Meta() SourceMeta
}

type Engine interface {
    Run(ctx context.Context, src Source, cfg Config) (*Result, error)
}
```

## Recommended error values

```go
var (
    ErrUnsupportedFormat   = errors.New("unsupported image format")
    ErrImageTooLarge       = errors.New("image exceeds configured limits")
    ErrInvalidConfig       = errors.New("invalid render config")
    ErrUnknownPalette      = errors.New("unknown palette preset")
    ErrUnknownMode         = errors.New("unknown render mode")
    ErrUnsupportedAlpha    = errors.New("unsupported alpha configuration")
    ErrReviewNotFound      = errors.New("review artifact not found")
)
```

## Decode and limits contract
`internal/ioimg` should expose functions roughly like this:

```go
package ioimg

type Limits struct {
    MaxWidth       int
    MaxHeight      int
    MaxPixels      int64
    MaxFileBytes   int64
}

type Decoded struct {
    Image  image.Image
    Meta   core.SourceMeta
}

func DecodeConfigAndValidate(r io.Reader, limits Limits) (core.SourceMeta, error)
func DecodeImage(r io.Reader, limits Limits) (*Decoded, error)
func EncodePNG(w io.Writer, img image.Image) error
```

Behavioral requirements:
- validate dimensions before full decode when possible
- reject unsupported or oversized files cleanly
- normalize format labels when reporting metadata

## Palette preset model

```go
package palette

type Preset struct {
    Key                string
    DisplayName        string
    Description        string
    Colors             []color.NRGBA
    RecommendedDither  core.DitherMode
    BrightnessAdjust   float64
    ContrastAdjust     float64
    GammaAdjust        float64
}

func AllPresets() []Preset
func MustGetPreset(key string) Preset
func GetPreset(key string) (Preset, bool)
```

## Palette extraction contract

```go
package palette

type ExtractOptions struct {
    Count         int
    GuidedPreset  *Preset
    PreserveBlack bool
}

func Extract(img image.Image, opts ExtractOptions) ([]color.NRGBA, error)
```

V1 note:
- public surface may only expose preset vs extract
- engine may internally support a guided-preset bias if that materially improves output quality

## Quantization contract

```go
package palette

type QuantizeOptions struct {
    Palette []color.NRGBA
    Dither  core.DitherMode
}

func QuantizeWholeImage(img image.Image, opts QuantizeOptions) (image.Image, error)
func QuantizeTile(img image.Image, palette []color.NRGBA, dither core.DitherMode) (image.Image, error)
```

Implementation requirements:
- deterministic palette ordering
- deterministic tie breaking on equal-distance color matches
- no random seeds in v1

## CGB background render contract

```go
package render

func RunRelaxed(ctx context.Context, src core.Source, cfg core.Config) (*core.Result, error)
func RunCGBBG(ctx context.Context, src core.Source, cfg core.Config) (*core.Result, error)
```

`RunCGBBG` requirements:
- tile size defaults to `8`
- colors per tile defaults to `4`
- max palettes defaults to `8`
- output metadata must include tile grid dimensions
- output metadata must include palette bank assignments

## Review bundle model
The review system exists for both browser review and reproducible testing.

## Review types

```go
package review

type ArtifactManifest struct {
    FinalPNG    string `json:"final_png"`
    PreviewPNG  string `json:"preview_png"`
    DebugPNG    string `json:"debug_png,omitempty"`
    MetaJSON    string `json:"meta_json"`
}

type Fingerprints struct {
    InputSHA256  string `json:"input_sha256"`
    ConfigSHA256 string `json:"config_sha256"`
    OutputSHA256 string `json:"output_sha256"`
}

type ReviewRecord struct {
    SchemaVersion   string                 `json:"schema_version"`
    ID              string                 `json:"id"`
    CreatedAt       time.Time              `json:"created_at"`
    Mode            string                 `json:"mode"`
    Config          core.Config            `json:"config"`
    Source          core.SourceMeta        `json:"source"`
    OutputWidth     int                    `json:"output_width"`
    OutputHeight    int                    `json:"output_height"`
    GlobalPalette   []string               `json:"global_palette,omitempty"`
    PaletteBanks    [][]string             `json:"palette_banks,omitempty"`
    TileAssignments []core.TileAssignment  `json:"tile_assignments,omitempty"`
    Artifacts       ArtifactManifest       `json:"artifacts"`
    Fingerprints    Fingerprints           `json:"fingerprints"`
    Metadata        map[string]any         `json:"metadata,omitempty"`
}
```

`schema_version` should be materialized explicitly in stored review JSON. Current stable value: `pixgbc.review/v1`.

## Review store interface

```go
package review

type Store interface {
    Save(ctx context.Context, record ReviewRecord, files map[string][]byte) error
    Get(ctx context.Context, id string) (ReviewRecord, error)
    OpenArtifact(ctx context.Context, id string, name string) (io.ReadSeekCloser, error)
    Delete(ctx context.Context, id string) error
    CleanupExpired(ctx context.Context, now time.Time) error
}
```

### V1 implementation
`TempStore` backed by a temp directory.

Requirements:
- review id is safe for URL/path usage
- artifacts stored under isolated review directory
- store cleanup callable on startup and optionally on interval

## CLI contract

## `convert`
Example:

```bash
pixgbc convert input.png -o out.png \
  --mode relaxed \
  --palette gbc-olive \
  --size 160x144 \
  --dither ordered \
  --scale 6 \
  --emit-review ./review
```

Flags:
- `--mode relaxed|cgb-bg`
- `--size WIDTHxHEIGHT`
- `--palette PRESET|extract`
- `--dither none|ordered|floyd-steinberg|atkinson`
- `--crop fit|fill`
- `--scale N`
- `--alpha flatten|reserve-color0`
- `--bg #RRGGBB`
- `--debug`
- `--emit-review PATH`
- `-o, --output PATH`

## `inspect`
Example:

```bash
pixgbc inspect input.png --json --debug
```

Output fields:
- width
- height
- alpha presence
- likely dominant colors
- recommended mode
- recommended palette preset
- optional debug thumbnail path

## `palette list`
Output:
- preset key
- display name
- short description
- palette swatches in text-friendly form (hex)

## Server HTTP contract

## Routes
- `GET /`
- `POST /api/render`
- `GET /api/renders/{id}`
- `GET /renders/{id}`
- `GET /artifacts/{id}/final.png`
- `GET /artifacts/{id}/preview.png`
- `GET /artifacts/{id}/debug.png`
- `GET /artifacts/{id}/meta.json`
- `GET /api/palettes`
- `GET /healthz`

## `POST /api/render`
Content type:
- `multipart/form-data`

Fields:
- `image` (required file)
- `mode`
- `palette`
- `width`
- `height`
- `crop`
- `dither`
- `scale`
- `alpha_mode`
- `bg_color`
- `debug`

Response: `200 OK`

```json
{
  "id": "rnd_123",
  "mode": "relaxed",
  "source": {
    "width": 1200,
    "height": 900,
    "has_alpha": false,
    "format": "png",
    "file_size": 182934,
    "frame_count": 1
  },
  "output": {
    "width": 160,
    "height": 144
  },
  "palette_strategy": "preset",
  "palette_preset": "gbc-olive",
  "global_palette": ["#0f380f", "#306230", "#8bac0f", "#9bbc0f"],
  "palette_banks": [],
  "artifacts": {
    "final_png": "/artifacts/rnd_123/final.png",
    "preview_png": "/artifacts/rnd_123/preview.png",
    "debug_png": "/artifacts/rnd_123/debug.png",
    "meta_json": "/artifacts/rnd_123/meta.json"
  },
  "fingerprints": {
    "input_sha256": "...",
    "config_sha256": "...",
    "output_sha256": "..."
  },
  "review_url": "/renders/rnd_123"
}
```

## `GET /api/renders/{id}`
Returns the saved `ReviewRecord` JSON.

## `GET /renders/{id}`
Human review page.

Page contents:
- original image
- normalized image (optional)
- final image
- preview image
- debug strip
- palette or palette banks
- tile heatmap in `cgb-bg` mode
- config block
- fingerprints block
- download links

## `GET /api/palettes`
Returns all presets.

Example:

```json
[
  {
    "key": "gbc-olive",
    "display_name": "GBC Olive",
    "description": "Default olive-green handheld look",
    "colors": ["#0f380f", "#306230", "#8bac0f", "#9bbc0f"],
    "recommended_dither": "ordered"
  }
]
```

## Security/operational flags for `serve`
- `--listen 127.0.0.1:8080`
- `--token VALUE` (optional if binding beyond localhost)
- `--artifact-ttl 24h`
- `--max-upload-bytes 10MB`

## JSON normalization requirements
For reproducibility:
- configs in review records must be normalized before hashing
- default values should be materialized explicitly in stored review JSON
- hex colors must use lowercase `#rrggbb`
- palette entries must be emitted in stable order

## Hashing rules
Hash:
- raw input bytes
- normalized config JSON bytes
- final PNG bytes

These hashes should be included in `ReviewRecord.Fingerprints`.

## Debug artifact rules
If `EmitDebug` is true, generate a debug sheet with panels such as:
- source
- normalized
- quantized
- final
- palette swatches
- tile-bank heatmap when applicable

If `EmitDebug` is false, omit the debug artifact entry or leave it empty.

## Testability requirements
Handlers should be constructed from injected dependencies:
- `core.Engine`
- `review.Store`
- server config

This allows direct `httptest` coverage without a real process-level listener.

## References
- Go `embed`: https://pkg.go.dev/embed
- Go `image`: https://pkg.go.dev/image
- Go `net/http/httptest`: https://pkg.go.dev/net/http/httptest
- Go `image/png`: https://pkg.go.dev/image/png
- Pan Docs / GBDev: https://gbdev.io/pandocs/
