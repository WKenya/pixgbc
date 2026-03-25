GO ?= go
CGO_ENABLED ?= 0

.PHONY: build test run-help

build:
	mkdir -p bin
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -o bin/pixgbc ./cmd/pixgbc

test:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test ./...

run-help:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) run ./cmd/pixgbc --help
