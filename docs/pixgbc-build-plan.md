# pixgbc Build Plan and Milestones

## Status
Draft v1

## Overall strategy
Build the smallest coherent product that already proves the core architecture:
- shared engine
- CLI surface
- local server
- review artifact system
- one strong default rendering path

Do **not** build every possible renderer or export up front.

## Phase 0: repository bootstrap
Goal: create the repo structure and basic developer ergonomics.

Deliverables:
- Go module initialized
- package structure created
- Makefile or task runner
- `README.md`
- sample input images for local testing
- `.gitignore`
- basic CI for `go test ./...`

Suggested folders:

```text
cmd/pixgbc
internal/app
internal/core
internal/source
internal/ioimg
internal/preprocess
internal/palette
internal/render
internal/review
internal/export
internal/web
web
samples
```

Definition of done:
- `go test ./...` passes on empty scaffolding
- binary boots with placeholder commands

## Phase 1: core types and safety
Goal: lock the engine shape and input validation.

Deliverables:
- `core.Config`, `core.Result`, `core.Source`, `core.Engine`
- decode-config path
- file size and dimension validation
- image normalization to `image.NRGBA`
- PNG encoding helper
- deterministic config normalization helper

Tasks:
1. define enums and defaults
2. implement config validation
3. implement input limits
4. add tests for oversized image rejection
5. add tests for config normalization

Definition of done:
- invalid inputs fail cleanly
- valid PNG/JPEG decode into normalized form
- config hashes are stable

## Phase 2: relaxed renderer MVP
Goal: ship the first useful image converter.

Deliverables:
- crop/pad support
- resize support
- tone adjustment support
- preset palette support
- palette extraction support
- whole-image quantization
- ordered dithering
- preview upscale

Tasks:
1. implement crop fill/fit helpers
2. implement resize helper
3. implement palette preset registry
4. implement palette extraction
5. implement quantization and nearest-color selection
6. implement ordered dithering
7. generate preview image
8. build golden-image fixtures

Definition of done:
- `relaxed` mode works end-to-end from file input to PNG output
- golden-image tests exist for at least 3 presets
- output is deterministic

## Phase 3: CLI surface
Goal: expose the engine through a usable CLI.

Deliverables:
- Cobra root command
- `convert`
- `inspect`
- `palette list`

Tasks:
1. wire config parsing from flags
2. implement `--size WIDTHxHEIGHT`
3. implement `--palette PRESET|extract`
4. implement `--debug`
5. implement `inspect` report
6. implement hex output for palette listing

Definition of done:
- CLI can convert images reproducibly
- CLI can inspect input images and emit useful metadata
- help text is clean and accurate

## Phase 4: review bundle system
Goal: create a common format for artifact inspection and regression testing.

Deliverables:
- `review.ReviewRecord`
- artifact manifest
- temp-disk review store
- hashing of input/config/output
- CLI `--emit-review`

Tasks:
1. define JSON review schema
2. implement artifact writing helpers
3. compute SHA-256 hashes
4. implement temp store layout
5. test reading/writing review bundles

Definition of done:
- CLI can emit a review directory with JSON + images
- stored metadata can be reloaded cleanly
- hashes are stable for identical runs

## Phase 5: local server + remote verification
Goal: turn the local server into a usable browser UI and test harness.

Deliverables:
- embedded static UI
- `POST /api/render`
- `GET /api/renders/{id}`
- `GET /renders/{id}`
- artifact download routes
- palette listing route
- health check route

Tasks:
1. embed static assets
2. implement multipart upload handler
3. save render artifacts into review store
4. implement HTML review page
5. implement LAN-safe server config options
6. add handler tests with `httptest`

Definition of done:
- image can be uploaded from a browser
- render result produces a review URL
- review URL can be opened remotely on the LAN
- review page exposes output, palette(s), config, and hashes

## Phase 6: strict `cgb-bg` renderer
Goal: add the stricter palette/tile-constrained mode.

Deliverables:
- tile grid splitter
- per-tile candidate palette solver
- palette-bank clustering
- tile-to-bank assignment
- per-tile requantization
- strict-mode metadata and debug visuals

Tasks:
1. implement fixed-grid tile walker
2. implement per-tile palette extraction
3. implement clustering into <= 8 palette banks
4. implement stable assignment rules
5. requantize each tile against its assigned bank
6. emit tile assignment metadata
7. generate tile-bank heatmap for debug sheet

Definition of done:
- `cgb-bg` mode produces stable outputs
- metadata includes palette banks and tile assignments
- browser review page can show tile-bank distribution

## Phase 7: polish and hardening
Goal: make the project durable and pleasant to use.

Deliverables:
- improved preset tuning
- better inspect recommendations
- TTL cleanup for old reviews
- upload/body limits on server
- benchmark coverage for hot paths
- improved docs and screenshots

Tasks:
1. tune presets using sample images
2. add server cleanup loop or startup cleanup
3. add request-size guardrails
4. add performance benchmarks
5. update README with screenshots and examples

Definition of done:
- project feels stable
- default outputs look consistently good
- local server is safe enough for LAN-only use

## Backlog after v1
Not part of the core release, but the architecture should leave room for them.

### Near-term extensions
- GIF input source
- frame-by-frame conversion
- batch conversion command
- palette-preview web page
- save/load named render presets

### Longer-term extensions
- public hosted service
- persistent review history
- shareable review links
- ROM-ish or tile-data export
- sprite-oriented mode

## Testing matrix

### Unit tests
- config validation
- aspect/crop helpers
- resize helpers
- alpha flattening
- palette extraction
- quantization tie-breaking
- cluster determinism
- review hashing

### Golden-image tests
- relaxed mode with preset palette
- relaxed mode with extracted palette
- strict mode with palette-bank clustering
- alpha flattening samples

### HTTP tests
- upload success
- upload rejection on oversized body
- render record retrieval
- missing artifact handling
- health route

### CLI integration tests
- convert writes expected files
- inspect emits expected report
- palette list prints expected presets
- `--emit-review` creates metadata + images

## Suggested implementation order inside the repo

### First week target
- Phases 0, 1, and part of 2
- enough code to decode, normalize, and resize images

### Second week target
- finish relaxed mode
- add CLI convert/inspect
- add first golden-image tests

### Third week target
- review bundle system
- local server API
- basic browser review page

### Fourth week target
- strict `cgb-bg` mode
- tile-bank heatmap/debug visuals
- documentation cleanup

## Release checklist for v1
- [ ] deterministic outputs verified
- [ ] golden-image tests checked in
- [ ] CLI help text finished
- [ ] review JSON schema stable
- [ ] local server review page working on LAN
- [ ] strict mode metadata exposed
- [ ] README includes screenshots and examples

## Example developer commands

```bash
go test ./...
go run ./cmd/pixgbc convert ./samples/input.png -o /tmp/out.png
go run ./cmd/pixgbc inspect ./samples/input.png --json
go run ./cmd/pixgbc serve --listen 127.0.0.1:8080
```

## Risks to watch

### 1. Output quality drift
Fix with golden-image tests and preset tuning.

### 2. Non-deterministic clustering
Fix with stable ordering, explicit tie-breaks, and deterministic test fixtures.

### 3. Scope creep
Fix by refusing to add animation, public hosting, and ROM export in the initial milestone set.

### 4. Local server becoming the product
Fix by keeping the server thin and routing everything through the shared engine.

### 5. Slow strict mode
Fix by optimizing only after correctness is proven.

## Success criteria
V1 is successful if:
- CLI conversion is reliable
- default relaxed outputs look strong
- strict mode produces visibly more form-constrained output
- browser review works from another machine on the LAN
- debug artifacts make output decisions easy to inspect
- code structure clearly supports later GIF support without a rewrite
