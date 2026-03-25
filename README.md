# pixgbc

`pixgbc` converts images into Game Boy Color-inspired pixel art.

Current implementation slice:

- Go module + package scaffold
- shared engine boundary
- `convert`, `inspect`, `palette list`, `serve`
- relaxed-mode MVP renderer
- embedded local web UI for single-image preview

Not done yet:

- strict `cgb-bg` renderer
- review bundle storage/URLs
- debug sheet export
- golden-image fixtures

## Commands

```sh
go run ./cmd/pixgbc --help
go run ./cmd/pixgbc palette list
go run ./cmd/pixgbc inspect --input path/to/input.png
go run ./cmd/pixgbc convert --input path/to/input.png --output out.png
go run ./cmd/pixgbc serve --addr 127.0.0.1:8080
```

## Build

```sh
make test
make build
```
