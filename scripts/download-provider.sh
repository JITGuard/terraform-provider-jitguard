#!/bin/bash
#
# Download a jitguard Terraform provider version from GitHub Releases and
# install it into the implicit filesystem-mirror location so `terraform init`
# picks it up without -plugin-dir.
#
# Implicit mirror path (Linux/macOS):
#   ~/.terraform.d/plugins/registry.terraform.io/barneyparker/jitguard/<version>/<os>_<arch>/terraform-provider-jitguard_v<version>
#
# Usage:
#   ./scripts/download-provider.sh [VERSION]
#
# Env:
#   REPO       GitHub owner/repo hosting the releases
#              (default barneyparker/terraform-provider-jitguard)
#   VERSION    provider version to install (default 0.1.0); pass as arg to override
#   OS_ARCH    explicit target, e.g. linux_amd64 (default: auto-detect)
set -euo pipefail

VERSION="${1:-${VERSION:-0.1.0}}"
REPO="${REPO:-barneyparker/terraform-provider-jitguard}"
RELEASE_BASE="https://github.com/${REPO}/releases/download/v${VERSION}"

HOSTNAME="registry.terraform.io"
NAMESPACE="barneyparker"
TYPE="jitguard"

detect_platform() {
  case "$(uname -s)-$(uname -m)" in
    Linux-x86_64)  echo "linux_amd64" ;;
    Linux-aarch64) echo "linux_arm64" ;;
    Darwin-x86_64) echo "darwin_amd64" ;;
    Darwin-arm64)  echo "darwin_arm64" ;;
    *) echo "unsupported platform: $(uname -s)-$(uname -m)" >&2; exit 1 ;;
  esac
}

OS_ARCH="${OS_ARCH:-$(detect_platform)}"
ZIP_NAME="terraform-provider-jitguard_v${VERSION}_${OS_ARCH}.zip"
ZIP_URL="${RELEASE_BASE}/${ZIP_NAME}"
SHA_URL="${RELEASE_BASE}/SHA256SUMS"

DEST_DIR="${XDG_CONFIG_HOME:-$HOME/.terraform.d}/plugins/$HOSTNAME/$NAMESPACE/$TYPE/$VERSION/$OS_ARCH"
PLUGIN="$DEST_DIR/terraform-provider-jitguard_v$VERSION"

echo "==> Downloading provider v${VERSION} (${OS_ARCH})"
echo "    ${ZIP_URL}"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

curl -fsSL -o "$TMPDIR/$ZIP_NAME" "$ZIP_URL"

# Verify the archive against the release's SHA256SUMS.
EXPECTED="$(curl -fsSL "$SHA_URL" | awk -v f="$ZIP_NAME" '$2 == f { print $1 }')"
if [ -z "$EXPECTED" ]; then
  echo "error: ${ZIP_NAME} not listed in SHA256SUMS — refusing unverified download" >&2
  exit 1
fi
ACTUAL="$(sha256sum "$TMPDIR/$ZIP_NAME" | awk '{ print $1 }')"
if [ "$ACTUAL" != "$EXPECTED" ]; then
  echo "error: checksum mismatch for ${ZIP_NAME}" >&2
  echo "  expected: $EXPECTED" >&2
  echo "  actual:   $ACTUAL" >&2
  exit 1
fi
echo "    verified sha256 checksum"

echo "==> Installing to ${DEST_DIR}"
mkdir -p "$DEST_DIR"
unzip -oq "$TMPDIR/$ZIP_NAME" -d "$TMPDIR/unpacked"
cp "$TMPDIR/unpacked/terraform-provider-jitguard" "$PLUGIN"
chmod +x "$PLUGIN"

echo "==> Installed: ${PLUGIN}"
echo "    terraform init will now find it automatically (implicit filesystem mirror)."
