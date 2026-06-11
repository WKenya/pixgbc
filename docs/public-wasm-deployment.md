# Public WASM Deployment

Read when: deploying pixgbc outside a trusted LAN, especially on Coolify or a VPS.

## Decision

Prefer a browser-local WASM deployment for public use.

The public site should be static assets only:

- `index.html`
- `styles.css`
- `app.js`
- `wasm-worker.js`
- `wasm_exec.js`
- `pixgbc.wasm`

In this mode, selected image files stay in the browser. The browser decodes the source image to RGBA, sends pixels to a Web Worker, and the worker runs the Go renderer compiled to WASM. The server never receives the source image bytes.

Keep `pixgbc serve` for private/admin use, LAN verification, and compatibility with review artifact URLs.

## Threat Model

Public server-side image processing has two risky inputs:

- compressed binary parsers: PNG/JPEG decoder surface
- compute/memory pressure: oversized or adversarial images, many concurrent renders

WASM static hosting removes both risks from the droplet. The droplet serves inert assets; the user's browser owns decode, memory use, and render CPU.

Residual browser-local risks:

- large files can use significant client memory
- slower mobile devices may render more slowly than native Go
- browser decoder support varies by format

Mitigation: cap local decoded inputs before sending pixels to WASM. Keep public UI to PNG/JPEG.

## Static Coolify Path

Build the static bundle:

```sh
make static-site
```

Docker build target:

```sh
docker build --target static -t pixgbc-static:local .
```

Coolify settings:

- Build Pack: Dockerfile
- Build target: `static`
- Exposed port: `8080`
- No persistent storage
- No app secrets required
- Resource limits: small CPU/memory are fine; the container only serves files

Recommended runtime hardening if using a Compose resource:

```yaml
services:
  pixgbc:
    build:
      context: .
      target: static
    expose:
      - "8080"
    read_only: true
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true
    pids_limit: 64
    mem_limit: 128m
    cpus: "0.25"
```

## Private Server Fallback

Use server mode only when artifact persistence or remote review URLs matter.

Recommended Coolify command:

```sh
serve --listen 0.0.0.0:8080 --artifact-ttl 1h --session-ttl 4h --request-rate-per-minute 60 --probe-rate-per-minute 10 --render-rate-per-minute 6 --max-concurrent-renders 1 --max-upload-bytes 4MB --max-source-width 4096 --max-source-height 4096 --max-source-pixels 16777216
```

Set `PIXGBC_TOKEN` as a runtime secret. The server reads it when `--token` is omitted.

Recommended Compose hardening:

```yaml
services:
  pixgbc:
    build: .
    environment:
      PIXGBC_TOKEN: ${PIXGBC_TOKEN:?}
    expose:
      - "8080"
    read_only: true
    tmpfs:
      - /tmp/pixgbc:rw,noexec,nosuid,nodev,size=128m
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true
    pids_limit: 64
    mem_limit: 512m
    cpus: "1.0"
```

DigitalOcean firewall:

- expose only `80/tcp`, `443/tcp`, and restricted SSH
- do not publish the app port directly to the host
- route through Coolify's proxy

## Operations

Hourly container restarts are optional cleanup, not the security boundary. Use static WASM for the public site. For server mode, use `tmpfs`, short artifact TTL, token auth, rate limits, concurrency limits, memory/CPU limits, and a read-only root filesystem.
