# pixgbc

`pixgbc` converts images into Game Boy Color-inspired pixel art.

Current implementation slice:

- Go module + package scaffold
- shared engine boundary
- `convert`, `inspect`, `palette list`, `serve`
- relaxed-mode MVP renderer
- strict `cgb-bg` tile/palette-bank renderer
- inspect recommendations for mode/palette fit
- composed debug-sheet export
- deterministic render golden-hash tests
- review bundle emission to temp/user-selected disk
- embedded local web UI with persisted review URLs/artifacts and basic render controls

Not done yet:

- binary/image fixture files for visual review

## Commands

```sh
go run ./cmd/pixgbc --help
go run ./cmd/pixgbc palette list
go run ./cmd/pixgbc inspect --input path/to/input.png --json
go run ./cmd/pixgbc convert --input path/to/input.png --output out.png
go run ./cmd/pixgbc convert --input path/to/input.png --output out.png --emit-review temp
go run ./cmd/pixgbc convert --input path/to/input.png --output out.png --mode cgb-bg --debug --emit-review temp
go run ./cmd/pixgbc serve --listen 127.0.0.1:8080 --artifact-ttl 24h --max-upload-bytes 10MB
```

`convert --emit-review` writes `final.png`, `preview.png`, and `meta.json` into a review bundle directory and prints the bundle path.

`convert --mode cgb-bg` runs the stricter tile/palette-bank solver. Add `--debug` to persist a composed debug sheet into the review bundle.

`inspect --json` now reports dominant colors, estimated strict-mode fit, and recommended mode/palette preset.

`serve` exposes browser controls for token, mode, preset vs extract, width/height, crop, dither, alpha mode, background color, preview scale, strict-mode tile params, and debug output.

If `serve` binds beyond localhost, `--token` is required. The token works via `?token=...` or `Authorization: Bearer ...`, and the web UI now propagates query-token links for protected review/artifact pages.

`serve` persists browser renders into a temp review store and exposes:

- `POST /api/render`
- `GET /api/renders/{id}`
- `GET /api/renders/{id}/artifacts/{name}`
- `GET /renders/{id}`

## Build

```sh
make test
make build
```
