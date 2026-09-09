#!/usr/bin/env bash
# Build a single-platform container image locally, mirroring the layout
# goreleaser's dockers_v2 uses (binary at <os>/<arch>/<name> in the context).
#
# Usage: scripts/docker-build.sh <image:tag> [goarch]
set -euo pipefail

IMAGE="${1:?usage: $0 <image:tag> [goarch]}"
GOARCH="${2:-$(go env GOARCH)}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CTX="$(mktemp -d)"
trap 'rm -rf "${CTX}"' EXIT

VERSION="$(git -C "${ROOT}" describe --tags --exact-match --match 'v*' 2>/dev/null || echo development)"
COMMIT="$(git -C "${ROOT}" rev-parse --short HEAD 2>/dev/null || echo unknown)"
MODULE="github.com/dntosas/kube-node-role-label"

mkdir -p "${CTX}/linux/${GOARCH}"
(
  cd "${ROOT}"
  CGO_ENABLED=0 GOOS=linux GOARCH="${GOARCH}" go build -trimpath \
    -ldflags="-s -w -X ${MODULE}/cmd.Version=${VERSION} -X ${MODULE}/cmd.CommitHash=${COMMIT}" \
    -o "${CTX}/linux/${GOARCH}/kube-node-role-label" .
)
cp "${ROOT}/Dockerfile" "${CTX}/Dockerfile"

docker build --platform "linux/${GOARCH}" -t "${IMAGE}" "${CTX}"
