# Repository Guidelines

## Project Structure & Module Organization
This repo is docs-first today. Product spec and build direction live in [`docs/pixgbc-spec-overview.md`](/Users/wesleykenyon/code/pixgbc/docs/pixgbc-spec-overview.md), [`docs/pixgbc-build-plan.md`](/Users/wesleykenyon/code/pixgbc/docs/pixgbc-build-plan.md), and [`docs/pixgbc-api-and-interfaces.md`](/Users/wesleykenyon/code/pixgbc/docs/pixgbc-api-and-interfaces.md). Implementation should follow the planned Go layout:

- `cmd/pixgbc` for the CLI entrypoint
- `internal/*` for engine, palette, render, review, and web packages
- `web/` for embedded static UI assets
- `samples/` for manual test fixtures once added

Keep new code aligned with the package boundaries defined in `docs/pixgbc-api-and-interfaces.md`.

## Build, Test, and Development Commands
Scaffold is not in place yet; contributors should add it with the first implementation pass. Standard commands for this repo should be:

- `go test ./...` runs the full Go test suite
- `go build ./cmd/pixgbc` builds the CLI binary
- `go run ./cmd/pixgbc --help` smoke-tests the command surface

If you add a `Makefile`, keep it thin and map targets directly to Go commands.

## Coding Style & Naming Conventions
Use idiomatic Go. Run `gofmt` on every edited file and prefer small packages with single responsibilities. Exported names use `CamelCase`; internal helpers use `camelCase`; CLI subcommands should mirror the spec (`convert`, `inspect`, `palette`, `serve`). Keep files under roughly 500 lines; split by concern when a package grows.

## Testing Guidelines
Use Go's standard `testing` package. Place tests next to implementation as `*_test.go`. Favor deterministic unit tests around config normalization, decode limits, palette behavior, and renderer output. For regressions, add fixture-driven or golden-image tests where output stability matters. Minimum gate for new code: `go test ./...`.

## Commit & Pull Request Guidelines
Current history is minimal, but repo policy requires Conventional Commits such as `feat: scaffold core packages` or `test: add quantizer golden fixtures`. PRs should include a short problem statement, implementation notes, test evidence, and screenshots when changing the future web UI or image outputs.

## Documentation & Contributor Notes
Read `docs/` before coding; this repo's architecture is specified there, not in `README.md` yet. Update the relevant doc when behavior, package boundaries, or command surfaces change.
