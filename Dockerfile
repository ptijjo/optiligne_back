# Build API Optiligne (Go) — image runtime pour Coolify / VPS.
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /out/api \
    ./cmd/api

# Image finale minimale
FROM alpine:3.21

RUN apk add --no-cache ca-certificates wget \
    && addgroup -S app \
    && adduser -S -G app -h /app app

WORKDIR /app

COPY --from=builder /out/api ./api
COPY --chown=app:app GTFS ./GTFS
COPY --chown=app:app data/perimetres ./data/perimetres

ENV APP_ENV=production \
    HTTP_PORT=8080 \
    GTFS_DATA_DIR=/app/GTFS \
    PERIMETRES_DIR=/app/data/perimetres

EXPOSE 8080

# Coolify injecte souvent PORT ; l'app lit HTTP_PORT puis PORT (défaut 8080).
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=5 \
    CMD wget --no-verbose --tries=1 --spider "http://127.0.0.1:${PORT:-${HTTP_PORT:-8080}}/health" || exit 1

USER app

CMD ["./api"]
