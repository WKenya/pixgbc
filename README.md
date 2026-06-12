# pixgbc

`pixgbc` converts images into Game Boy Color-inspired pixel art.

Current features:

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
- browser-local WASM/static web build for public no-upload deployment
- live WebSocket render progress in the browser UI
- deterministic sample generator + checked-in example inputs/outputs
- benchmark coverage for render/palette hot paths
- CLI integration coverage for help/convert/inspect/palette/review flows
- tracked docs/example assets under `docs/assets/`

## Photo Samples

Default:

![Default photo sample](.github/images/default-pea-relaxed.png)

Gray default:

![Gray default photo sample](.github/images/gray-default.png)

Warm gamma + contrast:

![Warm gamma and contrast photo sample](.github/images/cbg-extract-8-color-12-palatte.png)

`cgb-bg` no dither:

![CGB no dither photo sample](.github/images/cbg-no-dither.png)

## Commands

```sh
go run ./cmd/pixgbc --help
go run ./cmd/pixgbc palette list
go run ./cmd/pixgbc inspect --input path/to/input.png --json
go run ./cmd/pixgbc convert --input path/to/input.png --output out.png
go run ./cmd/pixgbc convert --input path/to/input.png --output out.png --emit-review temp
go run ./cmd/pixgbc convert --input path/to/input.png --output out.png --mode cgb-bg --debug --emit-review temp
go run ./cmd/pixgbc convert samples/portrait-alpha.png -o out.png --alpha flatten --bg '#f4f1e8'
go run ./cmd/pixgbc serve --listen 127.0.0.1:8080 --artifact-ttl 24h --max-upload-bytes 10MB
make static-site
make samples
make sample-outputs
make docs-assets
```

## Docker

Public/static build, preferred for internet deployment:

```sh
docker build --target static -t pixgbc-static:local .
docker run --rm -p 8080:8080 pixgbc-static:local
```

Server build, for private/admin review artifacts:

```sh
docker build -t pixgbc:local .
docker run --rm -p 8080:8080 -e PIXGBC_TOKEN=demo-token pixgbc:local
```

Run server mode with explicit hosted limits:

```sh
docker run --rm -p 8080:8080 -e PIXGBC_TOKEN=demo-token pixgbc:local serve --listen 0.0.0.0:8080 --artifact-ttl 1h --session-ttl 4h --request-rate-per-minute 60 --probe-rate-per-minute 10 --render-rate-per-minute 6 --max-concurrent-renders 1 --max-upload-bytes 4MB --max-source-width 4096 --max-source-height 4096 --max-source-pixels 16777216
```

Public deployment guidance: [docs/public-wasm-deployment.md](docs/public-wasm-deployment.md).

Factory/Coolify deployment intent lives in
[`deploy/factory.coolify.yml`](deploy/factory.coolify.yml). The GitLab pipeline
publishes the Dockerfile `static` target for Coolify; the `pixgbc.xyz`
Cloudflare route is staged in factory Terraform and enabled only through the
protected Cloudflare deploy gate.

`convert --emit-review` writes `source.png`, `final.png`, `preview.png`, `compare.png`, and `meta.json` into a review bundle directory and prints the bundle path.

`convert --mode cgb-bg` runs the stricter tile/palette-bank solver. Add `--debug` to persist a composed debug sheet into the review bundle.

`convert` also accepts `-o`, positional input, `--scale`, `--alpha`, `--bg`, `--brightness`, `--contrast`, `--gamma`, `--tile-size`, `--colors-per-tile`, and `--max-palettes`.

Checked-in sample inputs live in [samples/README.md](samples/README.md). `make sample-outputs` rebuilds the example PNG outputs and a strict-mode review bundle under `samples/`.

`inspect --json` now reports dominant colors, estimated strict-mode fit, and recommended mode/palette preset.

`serve` exposes browser controls for hosted sign-in, mode, preset vs extract, width/height, crop, dither, alpha mode, background color, brightness/contrast/gamma, preview scale, strict-mode tile params, and debug output.

Raw/debug affordances in the browser UI are now hidden by default on non-loopback hosts. Loopback shows them automatically; hosted/public sessions can toggle them with `Alt+Shift+D`.

In `cgb-bg`:

- `palette-mode preset` locks strict-mode banks back to the selected preset palette
- `palette-mode extract` uses direct sampled tile palettes from the image

If `serve` binds beyond localhost, `--token` is required by default. Use `--allow-open-access` to intentionally disable auth for a public/open demo. Browser sign-in exchanges the token for an `HttpOnly` session cookie; direct/manual access still works via `?token=...` or `Authorization: Bearer ...` when needed.

`serve` also accepts `PIXGBC_TOKEN` as a runtime environment fallback when `--token` is omitted.

Hosted hardening knobs:

- `--session-ttl 12h` controls browser session lifetime
- `--request-rate-per-minute 240` caps per-IP request volume across all routes; `0` disables
- `--probe-rate-per-minute 20` caps repeated suspicious path probes per IP; `0` disables
- `--render-rate-per-minute 60` caps per-IP render volume; `0` disables
- `--max-concurrent-renders 2` caps in-flight renders; `0` disables
- `--max-source-width 4096`, `--max-source-height 4096`, and `--max-source-pixels 16777216` cap decoded source dimensions

`serve` now sends basic hardening headers on all responses: CSP, `nosniff`, `DENY` framing, `no-referrer`, and locked-down permissions policy.

`serve` also drops common dotfile and encoded path-traversal probes with a flat `404` before they reach app routes.

`serve --artifact-ttl` now does an initial expired-artifact sweep at startup and keeps cleaning old review bundles on an interval while the server runs.

`serve` now logs startup cleanup, cleanup sweeps, HTTP requests, and render start/done events to stdout for easier local monitoring.

`serve` also keeps a session render history in the browser UI so you can jump back to recent review pages, previews, finals, records, and debug sheets without rerendering.

`serve` now streams staged render progress to the browser over WebSockets so the UI can show a classic loading bar during compute-heavy passes.

`serve` persists browser renders into a temp review store and exposes:

- `GET /api/session`
- `POST /api/session/login`
- `POST /api/session/logout`
- `GET /ws`
- `GET /api/palettes`
- `GET /api/renders`
- `POST /api/render`
- `GET /api/renders/{id}`
- `GET /api/renders/{id}/artifacts/{name}`
- `GET /renders/{id}`

Review bundles now also persist `source.png` and `compare.png`, so the web UI and docs pipeline can show original-vs-pixel comparisons without recomputing them.

The review page now renders palettes, config, hashes, and strict-mode tile-bank distribution directly in HTML in addition to raw JSON.

Review bundles now carry an explicit stable schema marker: `schema_version: "pixgbc.review/v1"`.

Release/ops docs:

- [docs/RELEASING.md](docs/RELEASING.md)
- [docs/LAN-VERIFICATION.md](docs/LAN-VERIFICATION.md)

## Examples

Relaxed preset render:

```sh
go run ./cmd/pixgbc convert samples/gradient-landscape.png -o /tmp/gradient.png --preview-out /tmp/gradient-preview.png --palette gbc-olive
```

![Relaxed render](docs/assets/gradient-relaxed.png)

![Relaxed compare](docs/assets/gradient-compare.png)

Alpha flattening with explicit background:

```sh
go run ./cmd/pixgbc convert samples/portrait-alpha.png -o /tmp/portrait.png --preview-out /tmp/portrait-preview.png --alpha flatten --bg '#f4f1e8'
```

![Alpha flatten render](docs/assets/portrait-alpha-relaxed.png)

![Alpha flatten compare](docs/assets/portrait-alpha-compare.png)

Strict `cgb-bg` render with debug sheet:

```sh
go run ./cmd/pixgbc convert samples/tile-banks.png -o /tmp/tile-banks.png --preview-out /tmp/tile-banks-preview.png --mode cgb-bg --debug --emit-review temp
```

![Strict render](docs/assets/tile-banks-cgb.png)

![Strict compare](docs/assets/tile-banks-compare.png)

![Strict debug sheet](docs/assets/tile-banks-debug.png)

## Build

```sh
make test
make build
make samples
make sample-outputs
make docs-assets
make bench
```
