# Releasing pixgbc

Read when: cutting a v1 build or doing final ship checks.

## Preflight

Run:

```sh
CGO_ENABLED=0 GOCACHE=/tmp/pixgbc-gocache make test
CGO_ENABLED=0 GOCACHE=/tmp/pixgbc-gocache make docs-assets
CGO_ENABLED=0 GOCACHE=/tmp/pixgbc-gocache make bench
```

Check:

- worktree clean
- sample docs assets regenerated under `docs/assets/`
- benchmark output has no obvious regressions
- README examples still match current CLI flags

## Manual checks

CLI:

```sh
./bin/pixgbc --help
./bin/pixgbc palette list
./bin/pixgbc inspect --input samples/gradient-landscape.png --json
./bin/pixgbc convert samples/gradient-landscape.png -o /tmp/pixgbc-gradient.png --preview-out /tmp/pixgbc-gradient-preview.png
./bin/pixgbc convert samples/tile-banks.png -o /tmp/pixgbc-cgb.png --preview-out /tmp/pixgbc-cgb-preview.png --mode cgb-bg --debug --emit-review temp
```

Server:

- follow [`docs/LAN-VERIFICATION.md`](/Users/wesleykenyon/code/pixgbc/docs/LAN-VERIFICATION.md)
- verify review page, artifact links, token flow, debug sheet

## Release notes minimum

Call out:

- relaxed + strict `cgb-bg` render modes
- CLI + local browser UI
- persisted review bundles with hashes/debug artifacts
- sample images + screenshots in `docs/assets/`

## Commit shape

Prefer small Conventional Commits:

- `feat: finalize v1 review schema`
- `docs: add release and LAN verification guides`
- `test: add cli integration coverage`

## Ship checklist

- [ ] `make test` clean
- [ ] `make docs-assets` clean
- [ ] `make bench` reviewed
- [ ] LAN review page checked from another device
- [ ] README screenshots/examples current
- [ ] release commit(s) created
