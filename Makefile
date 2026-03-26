GO ?= go
CGO_ENABLED ?= 0

.PHONY: build test run-help samples sample-outputs bench

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
	mkdir -p samples/outputs samples/reviews
	./bin/pixgbc convert samples/gradient-landscape.png -o samples/outputs/gradient-relaxed.png --preview-out samples/outputs/gradient-relaxed-preview.png --palette gbc-olive
	./bin/pixgbc convert samples/portrait-alpha.png -o samples/outputs/portrait-alpha-relaxed.png --preview-out samples/outputs/portrait-alpha-preview.png --alpha flatten --bg '#f4f1e8'
	./bin/pixgbc convert samples/tile-banks.png -o samples/outputs/tile-banks-cgb.png --preview-out samples/outputs/tile-banks-cgb-preview.png --mode cgb-bg --tile-size 8 --colors-per-tile 4 --max-palettes 8 --debug --emit-review samples/reviews

bench:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test -run '^$$' -bench . -benchmem ./internal/render ./internal/palette
