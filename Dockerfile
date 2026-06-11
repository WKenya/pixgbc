FROM golang:1.26.1-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY web ./web

ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/pixgbc ./cmd/pixgbc
RUN mkdir -p /out/static && \
    CGO_ENABLED=0 GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o /out/static/pixgbc.wasm ./cmd/pixgbc-wasm && \
    cp web/index.html web/app.js web/browser-mode.js web/server-mode.js web/styles.css web/wasm-client.js web/wasm-worker.js web/wasm_exec.js /out/static/

FROM busybox:1.37.0-musl AS static

WORKDIR /site
COPY --from=build /out/static /site
USER 10001:10001
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD wget -qO- http://127.0.0.1:8080/ >/dev/null || exit 1
CMD ["httpd", "-f", "-p", "8080", "-h", "/site"]

FROM alpine:3.22 AS server

RUN addgroup -S app && \
    adduser -S -D -H -u 10001 -G app appuser && \
    mkdir -p /tmp/pixgbc && \
    chown -R appuser:app /tmp/pixgbc

WORKDIR /app

COPY --from=build /out/pixgbc /usr/local/bin/pixgbc

ENV TMPDIR=/tmp/pixgbc

USER 10001:10001

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1

ENTRYPOINT ["pixgbc"]
CMD ["serve", "--listen", "0.0.0.0:8080", "--artifact-ttl", "1h", "--session-ttl", "4h", "--request-rate-per-minute", "60", "--probe-rate-per-minute", "10", "--render-rate-per-minute", "6", "--max-concurrent-renders", "1", "--max-upload-bytes", "4MB", "--max-source-width", "4096", "--max-source-height", "4096", "--max-source-pixels", "16777216"]
