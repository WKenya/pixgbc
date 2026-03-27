FROM golang:1.22.1-alpine AS build

WORKDIR /src

COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
COPY web ./web

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/pixgbc ./cmd/pixgbc

FROM alpine:3.20

RUN adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=build /out/pixgbc /usr/local/bin/pixgbc

ENV TMPDIR=/tmp/pixgbc

RUN mkdir -p /tmp/pixgbc && chown -R appuser:appuser /tmp/pixgbc /app

USER appuser

EXPOSE 8080

ENTRYPOINT ["pixgbc"]
CMD ["serve", "--listen", "0.0.0.0:8080", "--artifact-ttl", "24h", "--max-upload-bytes", "10MB"]
