#!/bin/bash
#
# Build the JIT Guard Terraform provider into a Provider Network Mirror
# layout and sync it to the dedicated mirror S3 bucket, served via CloudFront
# at https://jitguard.com/terraform/ so `terraform init` can install the
# provider over HTTPS before it is published to the Registry.
#
# Mirror protocol: https://developer.hashicorp.com/terraform/internals/provider-network-mirror-protocol
#
# Usage (CI or local):
#   VERSION=0.1.0 BUCKET=jitguard-prod-terraform-providers ./scripts/publish-mirror.sh
#
# Env:
#   VERSION      provider version (default 0.1.0)
#   BUCKET       S3 bucket to upload the mirror into (default jitguard-${ENVIRONMENT}-terraform-providers)
#   ENVIRONMENT  used for the default bucket name (default: TF_VAR_environment, else prod)
#   PLATFORMS    space-separated os_arch targets (default: linux_amd64 darwin_amd64 darwin_arm64 linux_arm64)
#   SKIP_UPLOAD  set to 1 to build the mirror layout without uploading
#   OUT          output dir (default: mktemp)
set -euo pipefail

VERSION="${VERSION:-0.1.0}"
ENVIRONMENT="${ENVIRONMENT:-${TF_VAR_environment:-prod}}"
BUCKET="${BUCKET:-jitguard-${ENVIRONMENT}-terraform-providers}"
PLATFORMS="${PLATFORMS:-linux_amd64 darwin_amd64 darwin_arm64 linux_arm64}"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
NS_DIR="registry.terraform.io/barneyparker/jitguard"
OUT="${OUT:-$(mktemp -d)}"
MIRROR="$OUT/$NS_DIR"
mkdir -p "$MIRROR"

declare -a KEYS=()
declare -a URLS=()
declare -a H1S=()

echo "==> Building provider v${VERSION} for: ${PLATFORMS}"

for plat in $PLATFORMS; do
  os="${plat%_*}"
  arch="${plat#*_}"
  name="terraform-provider-jitguard_${VERSION}_${os}_${arch}.zip"
  bin="$(mktemp -d)/terraform-provider-jitguard"

  echo "    ${os}_${arch}"
  (cd "$ROOT" && GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 \
    go build -trimpath -ldflags "-X main.version=${VERSION}" -o "$bin" .)

  # Zip the binary (bare name) so Terraform's unzip step yields the plugin.
  staged="$(mktemp -d)"
  cp "$bin" "$staged/terraform-provider-jitguard"
  (cd "$staged" && zip -q "$MIRROR/$name" terraform-provider-jitguard)

  # Terraform package hash v1: "h1:" + base64(sha256(zip)).
  h1="h1:$(openssl dgst -sha256 -binary "$MIRROR/$name" | openssl base64 -A | tr -d '\n')"
  KEYS+=("${os}_${arch}")
  URLS+=("$name")
  H1S+=("$h1")
done

# index.json — list available versions.
cat > "$MIRROR/index.json" <<EOF
{
  "versions": {
    "${VERSION}": {}
  }
}
EOF

# <version>.json — list installation packages per platform.
{
  echo "{"
  echo "  \"archives\": {"
  for i in "${!KEYS[@]}"; do
    sep=","
    if [ "$i" -eq $(( ${#KEYS[@]} - 1 )) ]; then sep=""; fi
    printf '    "%s": {\n      "url": "%s",\n      "hashes": ["%s"]\n    }%s\n' \
      "${KEYS[$i]}" "${URLS[$i]}" "${H1S[$i]}" "$sep"
  done
  echo "  }"
  echo "}"
} > "$MIRROR/${VERSION}.json"

echo "==> Mirror layout:"
find "$OUT" -type f | sort

if [ "${SKIP_UPLOAD:-}" = "1" ]; then
  echo "==> SKIP_UPLOAD=1 — layout written to ${OUT} (not uploaded)"
  exit 0
fi

echo "==> Syncing to s3://${BUCKET}/terraform/ ..."
aws s3 sync "$OUT/" "s3://${BUCKET}/terraform/" \
  --exclude "*" \
  --include "registry.terraform.io/*" \
  --cache-control "public, max-age=300"

echo "==> Syncing docs to s3://${BUCKET}/terraform/docs/ ..."
if [ -d "$ROOT/docs" ]; then
  aws s3 sync "$ROOT/docs/" "s3://${BUCKET}/terraform/docs/" \
    --cache-control "public, max-age=300"
  if [ -f "$ROOT/local-usage.md" ]; then
    aws s3 cp "$ROOT/local-usage.md" "s3://${BUCKET}/terraform/docs/local-usage.md" \
      --cache-control "public, max-age=300"
  fi
else
  echo "    (no docs/ directory — run tfplugindocs generate first)"
fi

echo "==> Done. Mirror base URL: https://jitguard.com/terraform/"
echo "    Docs: https://jitguard.com/terraform/docs/"
