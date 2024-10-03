#!/usr/bin/env sh

# Descargar arelo de https://github.com/makiuchi-d/arelo
arelo -p "**/*.go" -d 500ms  -- go run cmd/server/main.go
