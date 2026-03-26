# LAN Verification

Read when: validating `pixgbc serve` from another machine on the same network.

## Start server

Use a non-local bind. Token required.

```sh
CGO_ENABLED=0 GOCACHE=/tmp/pixgbc-gocache make build
./bin/pixgbc serve --listen 0.0.0.0:8080 --token demo-token --artifact-ttl 24h --max-upload-bytes 10MB
```

Find host IP on the serving machine:

```sh
ipconfig getifaddr en0
```

Example review URL base:

```text
http://192.168.1.23:8080/?token=demo-token
```

## Remote checks

From another device on the same LAN:

1. Open the base URL with `?token=demo-token`.
2. Confirm the UI loads and palettes populate.
3. Upload `samples/gradient-landscape.png` or another PNG/JPEG.
4. Render relaxed mode.
5. Open the generated review page.
6. Confirm `preview.png`, `final.png`, and `record.json` links work.
7. Render strict `cgb-bg` mode with debug enabled.
8. Confirm the review page shows palette banks, fingerprints, and tile-bank distribution.
9. Open the debug sheet link.
10. Confirm links preserve `?token=demo-token`.

## Failure checks

Verify:

- omitting token returns `401` on `/api/render`
- oversized upload returns `400`
- invalid `bg_color` or `gamma` returns `400`

## Cleanup check

For a short TTL smoke:

```sh
./bin/pixgbc serve --listen 127.0.0.1:8080 --artifact-ttl 1m
```

Create a render, wait past TTL, restart server, confirm expired review bundle is gone.
