---
name: pixgbc-cli
description: >-
  Use when the user wants to operate pixgbc from the terminal: inspect inputs,
  list palettes, convert images, compare relaxed vs cgb-bg output, emit review
  bundles, or run the local web server.
---

# pixgbc CLI

Use `./bin/pixgbc` if the binary exists. Else use `go run ./cmd/pixgbc`.

## Fast paths

- List palettes:
  ```sh
  ./bin/pixgbc palette list
  ```
- Inspect an image:
  ```sh
  ./bin/pixgbc inspect --input path/to/input.png --json
  ```
- Convert one image:
  ```sh
  ./bin/pixgbc convert path/to/input.png -o /tmp/out.png --preview-out /tmp/out-preview.png
  ```
- Emit a review bundle:
  ```sh
  ./bin/pixgbc convert path/to/input.png -o /tmp/out.png --preview-out /tmp/out-preview.png --emit-review temp
  ```
- Start local web UI:
  ```sh
  ./bin/pixgbc serve --listen 127.0.0.1:8080
  ```

## Mode choice

- `relaxed`: default; better-looking, less constrained
- `cgb-bg`: stricter tile/palette-bank render

Strict mode palette behavior:
- `--palette-mode extract`: source-driven; recommended default for `cgb-bg`
- `--palette-mode preset`: lock strict banks to selected preset

Example strict render:

```sh
./bin/pixgbc convert path/to/input.png \
  -o /tmp/cgb.png \
  --preview-out /tmp/cgb-preview.png \
  --mode cgb-bg \
  --palette-mode extract \
  --emit-review temp
```

## Compare workflow

Use `--emit-review temp` or a fixed directory. Review bundles include:

- `source.png`
- `final.png`
- `preview.png`
- `compare.png`
- `meta.json`
- `debug.png` when debug enabled

For mode comparisons, run twice into separate review dirs:

```sh
./bin/pixgbc convert samples/gradient-landscape.png \
  -o /tmp/relaxed.png \
  --preview-out /tmp/relaxed-preview.png \
  --mode relaxed \
  --emit-review /tmp/reviews/relaxed

./bin/pixgbc convert samples/gradient-landscape.png \
  -o /tmp/cgb.png \
  --preview-out /tmp/cgb-preview.png \
  --mode cgb-bg \
  --palette-mode extract \
  --emit-review /tmp/reviews/cgb
```

Then compare the two `compare.png` cards or `final.png` outputs.

## Crop tip

- `--crop fill`: fills frame; center-crops
- `--crop fit`: preserves whole image; may letterbox

If the user says the render only shows the middle, switch to:

```sh
--crop fit
```

## Sample inputs in this repo

- `samples/gradient-landscape.png`
- `samples/portrait-alpha.png`
- `samples/tile-banks.png`

Do not use `.github/images/*.png` as source examples for conversion. Those are already-rendered demo outputs.

## Useful flags

- `--palette <name>`
- `--palette-mode preset|extract`
- `--mode relaxed|cgb-bg`
- `--crop fill|fit`
- `--dither ordered|none|floyd-steinberg|atkinson`
- `--brightness <n>`
- `--contrast <n>`
- `--gamma <n>`
- `--alpha flatten|reserve-color0`
- `--bg '#f4f1e8'`
- `--tile-size <n>`
- `--colors-per-tile <n>`
- `--max-palettes <n>`
- `--debug`
- `--emit-review temp|PATH`

## Good defaults

- Photos:
  ```sh
  ./bin/pixgbc convert samples/gradient-landscape.png -o /tmp/out.png --preview-out /tmp/out-preview.png --crop fit
  ```
- Strict/source-driven:
  ```sh
  ./bin/pixgbc convert samples/tile-banks.png -o /tmp/out.png --preview-out /tmp/out-preview.png --mode cgb-bg --palette-mode extract --emit-review temp
  ```
- Alpha source:
  ```sh
  ./bin/pixgbc convert samples/portrait-alpha.png -o /tmp/out.png --preview-out /tmp/out-preview.png --alpha flatten --bg '#f4f1e8'
  ```

## Verify

If the user asks to test the app:

```sh
GOCACHE=/tmp/pixgbc-gocache go test ./internal/app ./internal/web
GOCACHE=/tmp/pixgbc-gocache go build -o ./bin/pixgbc ./cmd/pixgbc
```
