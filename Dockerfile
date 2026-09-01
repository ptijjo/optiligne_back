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

RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app -h /app app

WORKDIR /app

COPY --from=builder /out/api ./api
COPY --chown=app:app GTFS ./GTFS
COPY --chown=app:app data/perimetres ./data/perimetres

# Coolify injecte parfois PORT : garder HTTP_PORT et PORT identiques (9191 ici).
# Healthcheck : configurer dans l'UI Coolify (GET /health), pas dans l'image.
ENV APP_ENV=production \
    PORT=9191 \
    HTTP_PORT=9191 \
    GTFS_DATA_DIR=/app/GTFS \
    PERIMETRES_DIR=/app/data/perimetres

EXPOSE 9191

USER app

CMD ["./api"]
