#!/usr/bin/env sh

# Descargar arelo de https://github.com/makiuchi-d/arelo
arelo -p "**/*.go"  -d 600ms  -- go run cmd/client/main.go
