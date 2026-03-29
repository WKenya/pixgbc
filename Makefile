GO ?= go
CGO_ENABLED ?= 0

.PHONY: build test run-help samples sample-outputs docs-assets bench

build:
	mkdir -p bin
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -o bin/pixgbc ./cmd/pixgbc

test:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test ./...

run-help:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) run ./cmd/pixgbc --help

samples:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) run ./cmd/pixgbc-samplegen

sample-outputs: build samples
	mkdir -p samples/outputs samples/reviews/gradient samples/reviews/portrait samples/reviews/tile-banks
	./bin/pixgbc convert samples/gradient-landscape.png -o samples/outputs/gradient-relaxed.png --preview-out samples/outputs/gradient-relaxed-preview.png --palette gbc-olive --emit-review samples/reviews/gradient
	./bin/pixgbc convert samples/portrait-alpha.png -o samples/outputs/portrait-alpha-relaxed.png --preview-out samples/outputs/portrait-alpha-preview.png --alpha flatten --bg '#f4f1e8' --emit-review samples/reviews/portrait
	./bin/pixgbc convert samples/tile-banks.png -o samples/outputs/tile-banks-cgb.png --preview-out samples/outputs/tile-banks-cgb-preview.png --mode cgb-bg --tile-size 8 --colors-per-tile 4 --max-palettes 8 --debug --emit-review samples/reviews/tile-banks

docs-assets: sample-outputs
	mkdir -p docs/assets
	cp samples/outputs/gradient-relaxed.png docs/assets/gradient-relaxed.png
	cp samples/outputs/portrait-alpha-relaxed.png docs/assets/portrait-alpha-relaxed.png
	cp samples/outputs/tile-banks-cgb.png docs/assets/tile-banks-cgb.png
	cp "$$(find samples/reviews/gradient -maxdepth 1 -mindepth 1 -type d | sort | tail -n 1)/compare.png" docs/assets/gradient-compare.png
	cp "$$(find samples/reviews/portrait -maxdepth 1 -mindepth 1 -type d | sort | tail -n 1)/compare.png" docs/assets/portrait-alpha-compare.png
	cp "$$(find samples/reviews/tile-banks -maxdepth 1 -mindepth 1 -type d | sort | tail -n 1)/compare.png" docs/assets/tile-banks-compare.png
	cp "$$(find samples/reviews/tile-banks -maxdepth 1 -mindepth 1 -type d | sort | tail -n 1)/debug.png" docs/assets/tile-banks-debug.png

bench:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test -run '^$$' -bench . -benchmem ./internal/render ./internal/palette
