# pixgbc Specification Overview

## Status
Draft v1

## Purpose
`pixgbc` is a Go-based image conversion tool with both a CLI and a locally hosted web UI. Its job is to convert source images into Game Boy Color–inspired pixel art using a palette-first rendering pipeline.

The product is **not** a ROM builder or hardware emulator. The product goal is:

1. generate visually strong retro outputs for still images
2. support a stricter tile/palette-constrained mode that feels closer to CGB background art
3. expose the same engine through both CLI and local server
4. provide a strong local verification/testing interface for reviewing output from another device on the same network

## Product goals

### Primary goals
- Produce good-looking 8-bit / Game Boy Color–inspired still images.
- Support both a relaxed palette-driven mode and a stricter CGB-background-like mode.
- Make the CLI the core entry point.
- Bundle a lightweight browser UI into the same binary for local self-hosting.
- Make remote verification easy through stable URLs, metadata, and downloadable artifacts.

### Non-goals for v1
- No GIF/video conversion in the user-facing feature set.
- No ROM export.
- No public SaaS deployment.
- No accounts, auth system, billing, or database.
- No true hardware-accurate emulation.
- No sprite-specific export pipeline.

## Product shape
One binary:

- `pixgbc convert`
- `pixgbc inspect`
- `pixgbc palette`
- `pixgbc serve`

The web UI is embedded into the same Go binary and served locally.

## Design principles

### 1. Shared engine first
The rendering engine must be a shared package with zero awareness of CLI flags or HTTP handlers.

### 2. Palette-first, not style-first
The product should be organized around palette policies and rendering constraints, not vague “style filters.”

### 3. Best-looking over strict purity
Strict mode should still optimize for strong output. Hard constraints are useful only when they visibly improve authenticity or reviewability.

### 4. Deterministic output
Given the same input bytes and the same config, the tool should generate the same output bytes and the same review metadata.

### 5. Reviewability is a feature
The local server is not just a GUI. It is a repeatable verification surface with stable artifact URLs, hashes, and debug views.

## Supported modes

## `relaxed`
Default mode.

Behavior:
- palette-driven whole-image quantization
- optional preset palette or auto extraction
- optional dithering
- target output defaults to `160x144`
- tuned for the best visual output

Use cases:
- screenshots
- profile images
- posters
- general retro conversion

## `cgb-bg`
Stricter background-like mode.

Behavior:
- image resized to output canvas, default `160x144`
- split into `8x8` tiles
- each tile quantized against 4 colors
- per-tile candidate palettes clustered into up to 8 shared background palette banks
- tiles assigned to the nearest bank
- output remains visually tuned rather than rigidly emulator-perfect

Use cases:
- “closer to the form” output
- comparing palette-bank distribution
- game-art-adjacent experiments

## Default settings

### Global defaults
- mode: `relaxed`
- output size: `160x144`
- crop mode: `fill`
- preview scale: `6`
- alpha mode: `flatten`
- dither: `ordered`
- palette preset: `gbc-olive`

### Strict-mode defaults
- tile size: `8`
- colors per tile: `4`
- max palettes: `8`

## Supported palette strategies

### Preset
Use a built-in curated palette exactly.

### Extract
Auto-extract a palette from the source image and use it as the master palette input for conversion.

### Extract-guided (recommended internal extension)
This may still be expressed as `extract` in the public UI for v1, but the engine should be able to bias extracted colors toward a preset family.

Reason: raw extraction often looks technically correct but aesthetically weak.

## Preset palette list
Initial preset set:

- `dmg-pea`
- `dmg-gray`
- `gbc-olive`
- `gbc-pocket`
- `lcd-cool`
- `warm-backlight`

Each preset should define:
- a palette
- recommended default dither
- recommended tone adjustments
- a display label and short description

## Alpha handling
Support narrow alpha handling in v1.

### `flatten`
Default.
Composite the image onto a background color before quantization.

### `reserve-color0`
Optional behavior.
Reserve palette entry 0 for transparent/empty pixels when that is useful for future export paths.

For full-image conversion, `flatten` should remain the default.

## Architecture

```text
cmd/pixgbc/
  main.go

internal/app/
  root.go
  convert_cmd.go
  inspect_cmd.go
  palette_cmd.go
  serve_cmd.go

internal/core/
  config.go
  types.go
  errors.go

internal/source/
  source.go
  single_image.go

internal/ioimg/
  decode.go
  encode.go
  limits.go

internal/preprocess/
  normalize.go
  crop.go
  resize.go
  tone.go
  alpha.go

internal/palette/
  preset.go
  extract.go
  quantize.go
  distance.go
  rgb555.go
  cluster.go

internal/render/
  relaxed.go
  cgb_bg.go

internal/review/
  types.go
  store.go
  temp_store.go
  hashes.go

internal/export/
  png.go
  json.go
  debug_sheet.go

internal/web/
  server.go
  handlers.go
  dto.go
  views.go

web/
  index.html
  app.js
  styles.css
```

## Shared rendering pipeline
Both modes should use the same high-level stages.

1. image config read and limits check
2. full decode
3. normalize image into `image.NRGBA`
4. alpha handling
5. crop/pad to requested aspect
6. resize to target canvas
7. light tonal conditioning
8. solve palette(s)
9. quantize
10. dither
11. export final, preview, and metadata
12. store review artifacts when requested by the server or CLI

## Mode-specific pipelines

### Relaxed pipeline
1. decode and normalize
2. crop or pad
3. resize to target canvas
4. tonal adjustment
5. solve one global palette
6. quantize whole image
7. apply dither
8. upscale preview with nearest-neighbor
9. emit metadata/debug artifacts

### CGB background pipeline
1. decode and normalize
2. crop or pad
3. resize to target canvas
4. split into `8x8` tiles
5. derive a candidate 4-color palette per tile
6. cluster tile palettes into <= 8 banks
7. assign each tile to a bank
8. requantize each tile against its assigned bank
9. optionally apply mild ordered dithering
10. compose final image
11. upscale preview with nearest-neighbor
12. emit palette bank metadata and tile assignments

## Local server goals
The local server must support:
- running conversions from a browser
- viewing artifacts from another device on the LAN
- deterministic review metadata
- stable artifact URLs
- download of final/debug/metadata files

The local server should be considered both a UX feature and a testing harness.

## Review artifact model
Every render run should be representable as a review bundle with:
- run id
- original source metadata
- normalized config
- selected palette or palette banks
- fingerprints (input/config/output hashes)
- links to final image, preview image, debug image, and metadata JSON

This review bundle should be usable by:
- local browser UI
- CLI `--emit-review`
- test fixtures

## Remote verification design
A successful render should produce a review page reachable at:

- `/renders/{id}`

That page should show:
- original image
- resized or normalized image
- final output
- debug strip
- selected palette(s)
- tile-bank heatmap in `cgb-bg` mode
- full config used
- input/config/output hashes
- download links

This is the main remote verification interface.

## LAN usage
By default:
- bind to `127.0.0.1:8080`

Optional:
- bind to `0.0.0.0:8080` for LAN access

If bound beyond localhost, the server should support a lightweight access token option in v1 to avoid accidental exposure.

## Command overview

### `pixgbc convert`
Convert one image and write output PNG.

Optional flags should allow emitting review artifacts and metadata.

### `pixgbc inspect`
Analyze an image and report:
- source dimensions
- alpha presence
- likely dominant colors
- recommended mode
- recommended preset
- optional debug thumbnail sheet

### `pixgbc palette list`
List available presets and descriptions.

### `pixgbc serve`
Run the local self-hosted server with UI + JSON API + review artifact endpoints.

## Performance expectations
This is primarily a CPU-bound image-processing tool.

Optimization priorities:
1. correctness
2. deterministic outputs
3. acceptable latency for still images
4. memory safety for uploaded files

Avoid premature complexity such as GPU offload or distributed job processing.

## Security and safety for local server
Minimum safeguards for v1:
- request body limits
- image dimension limits
- read config before full decode
- safe temp storage
- cleanup of review artifacts via TTL or process shutdown
- optional access token when bound to non-localhost addresses
- no execution of user-supplied code

## Testing strategy summary
Must-have test categories:
- golden-image tests
- deterministic palette tests
- crop/resize tests
- alpha flattening tests
- review artifact metadata tests
- HTTP handler tests
- oversized image rejection tests

## Milestone shape
Recommended order:
1. engine skeleton and file safety
2. relaxed mode
3. CLI surface
4. review bundle format
5. local server and review UI
6. strict `cgb-bg` mode
7. regression fixtures and polish

## References
- Pan Docs / GBDev: https://gbdev.io/pandocs/
- Go `embed`: https://pkg.go.dev/embed
- Go `image`: https://pkg.go.dev/image
- Go `image/png`: https://pkg.go.dev/image/png
- Go `image/gif`: https://pkg.go.dev/image/gif
- Go `net/http/httptest`: https://pkg.go.dev/net/http/httptest
- Go `golang.org/x/image/draw`: https://pkg.go.dev/golang.org/x/image/draw
- Cobra: https://github.com/spf13/cobra
