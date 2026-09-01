#!/bin/sh
# Même priorité que config.LoadFromEnv : HTTP_PORT puis PORT.
port="${HTTP_PORT:-${PORT:-9191}}"
wget --no-verbose --tries=1 --spider "http://127.0.0.1:${port}/health"
