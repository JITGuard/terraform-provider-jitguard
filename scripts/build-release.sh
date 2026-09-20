#!/bin/bash
#
# Build the JIT Guard Terraform provider release artifacts: one zip per
# platform (bare binary inside) plus a SHA256SUMS file, ready to attach to a
# GitHub Release.
#
# Usage:
#   ./scripts/build-release.sh <version>          # e.g. 0.2.0 (no leading v)
#
# Env:
#   PLATFORMS  space-separated os_arch targets
#              (default: linux_amd64 linux_arm64 darwin_amd64 darwin_arm64)
#   OUT        output dir (default: <repo>/dist)
set -euo pipefail

VERSION="${1:?usage: build-release.sh <version>}"
PLATFORMS="${PLATFORMS:-linux_amd64 linux_arm64 darwin_amd64 darwin_arm64}"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${OUT:-$ROOT/dist}"
rm -rf "$OUT"
mkdir -p "$OUT"

echo "==> Building provider v${VERSION} for: ${PLATFORMS}"

for plat in $PLATFORMS; do
  os="${plat%_*}"
  arch="${plat#*_}"
  name="terraform-provider-jitguard_v${VERSION}_${os}_${arch}.zip"
  bin="$(mktemp -d)/terraform-provider-jitguard"

  echo "    ${os}_${arch}"
  (cd "$ROOT" && GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 \
    go build -trimpath -ldflags "-X main.version=${VERSION}" -o "$bin" .)

  # Zip the binary with its bare name so Terraform's unzip step yields the plugin.
  staged="$(mktemp -d)"
  cp "$bin" "$staged/terraform-provider-jitguard"
  (cd "$staged" && zip -q "$OUT/$name" terraform-provider-jitguard)
done

(cd "$OUT" && sha256sum ./*.zip > SHA256SUMS)

echo "==> Artifacts in ${OUT}:"
ls -1 "$OUT"
