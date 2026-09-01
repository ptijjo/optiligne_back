#!/bin/sh
set -e

echo "Import GTFS (no-op si déjà à jour)..."
./importer

echo "Démarrage API..."
exec ./api
