#!/usr/bin/env bash

# Builds portable, optimized lazygitk binaries into dist/<os>_<arch>/.
# Usage: scripts/build_dist.sh [os/arch ...]
#   e.g. scripts/build_dist.sh linux/arm64 darwin/arm64
# With no arguments, builds linux/amd64 and windows/amd64. Builds use a
# temporary GOCACHE that is removed on exit (including Ctrl-C), so no build
# cache is left behind. Binaries are built with buildSource=forkDist, which
# disables upstream's self-updater (see pkg/updates/updates.go).

set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(dirname "$script_dir")

cd "$repo_root"

if [ "$#" -eq 0 ]; then
  set -- linux/amd64 windows/amd64
fi

for target in "$@"; do
  if [[ ! "$target" =~ ^[^/]+/[^/]+$ ]]; then
    echo "Error: invalid target '$target'; expected os/arch, e.g. linux/arm64" >&2
    exit 1
  fi
done

# Exit (rather than just dying on the signal) so that the EXIT trap runs.
trap 'exit 130' INT
trap 'exit 143' TERM

gocache=$(mktemp -d)
trap 'rm -rf "$gocache"' EXIT

export GOCACHE="$gocache"
export GOTOOLCHAIN=local
export CGO_ENABLED=0
export GOFLAGS=-mod=vendor

version=$(git describe --tags --always --dirty 2>/dev/null) || version=unversioned
commit=$(git rev-parse HEAD 2>/dev/null) || commit=
build_date=$(date -u +%Y-%m-%dT%H:%M:%SZ)

ldflags="-s -w -X main.version=$version -X main.commit=$commit -X main.date=$build_date -X main.buildSource=forkDist"

for target in "$@"; do
  goos=${target%%/*}
  goarch=${target#*/}
  binary=lazygitk
  if [ "$goos" = windows ]; then
    binary=lazygitk.exe
  fi
  out="dist/${goos}_${goarch}/${binary}"

  mkdir -p "$(dirname "$out")"
  echo "Building $out"
  if ! GOOS="$goos" GOARCH="$goarch" go build -trimpath -ldflags "$ldflags" -o "$out" .; then
    echo "Error: build failed for $target" >&2
    exit 1
  fi
  echo "Built $out"
done
