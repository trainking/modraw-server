#!/usr/bin/env bash
# Build modraw-server for Linux deployment.
# Run this from the project root before running the Ansible playbook.
#
# Usage:
#   bash deploy/build.sh

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUTPUT_DIR="${PROJECT_ROOT}/deploy/bin"

echo "[build] Project root: ${PROJECT_ROOT}"
echo "[build] Output dir:   ${OUTPUT_DIR}"

mkdir -p "${OUTPUT_DIR}"

cd "${PROJECT_ROOT}"

echo "[build] Downloading dependencies..."
go mod download

echo "[build] Cross-compiling for linux/amd64..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o "${OUTPUT_DIR}/modraw-server" ./cmd/server

echo "[build] Binary size: $(du -h "${OUTPUT_DIR}/modraw-server" | cut -f1)"
echo "[build] Done: ${OUTPUT_DIR}/modraw-server"
