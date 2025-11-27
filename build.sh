#!/usr/bin/env bash
set -euo pipefail

# You can override VERSION from the environment, otherwise it defaults to "dev".
VERSION="${VERSION:-dev}"

# Short git commit (or "local" if not in a git repo).
GIT_COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo "local")"

# UTC build time in ISO-8601 format.
BUILD_TIME="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

echo "Building stratalog..."
echo "  VERSION    = ${VERSION}"
echo "  GIT_COMMIT = ${GIT_COMMIT}"
echo "  BUILD_TIME = ${BUILD_TIME}"

go build -ldflags "\
  -X github.com/dalemusser/stratalog/internal/app/system/versioninfo.Version=${VERSION} \
  -X github.com/dalemusser/stratalog/internal/app/system/versioninfo.GitCommit=${GIT_COMMIT} \
  -X github.com/dalemusser/stratalog/internal/app/system/versioninfo.BuildTime=${BUILD_TIME}" \
  -o stratalog ./cmd/stratalog


