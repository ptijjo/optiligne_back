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

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /out/importer \
    ./cmd/importer

# Image finale minimale
FROM alpine:3.21

RUN apk add --no-cache ca-certificates wget \
    && addgroup -S app \
    && adduser -S -G app -h /app app

WORKDIR /app

COPY --from=builder /out/api ./api
COPY --from=builder /out/importer ./importer
COPY --chown=app:app GTFS ./GTFS
COPY --chown=app:app data/perimetres ./data/perimetres
COPY docker/healthcheck.sh /usr/local/bin/healthcheck.sh
COPY docker/entrypoint.sh ./entrypoint.sh
RUN chmod +x /usr/local/bin/healthcheck.sh ./entrypoint.sh

# Coolify injecte parfois PORT : garder HTTP_PORT et PORT identiques (9191 ici).
ENV APP_ENV=production \
    PORT=9191 \
    HTTP_PORT=9191 \
    GTFS_DATA_DIR=/app/GTFS \
    PERIMETRES_DIR=/app/data/perimetres

EXPOSE 9191

# L’entrypoint lance l’import GTFS avant l’API. Premier import (shapes) :
# plusieurs minutes. Coolify rollback si unhealthy pendant ce délai.
HEALTHCHECK --interval=30s --timeout=5s --start-period=900s --retries=3 \
    CMD /usr/local/bin/healthcheck.sh

USER app

CMD ["./entrypoint.sh"]
