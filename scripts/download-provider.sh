#!/bin/bash
#
# Download a jitguard Terraform provider version from the network mirror
# and install it into the implicit filesystem-mirror location so `terraform
# init` picks it up without -plugin-dir.
#
# Implicit mirror path (Linux/macOS):
#   ~/.terraform.d/plugins/registry.terraform.io/barneyparker/jitguard/<version>/<os>_<arch>/terraform-provider-jitguard_v<version>
#
# Usage:
#   ./scripts/download-provider.sh [VERSION]
#
# Env:
#   MIRROR_URL    mirror base URL (default https://jitguard.com/terraform/)
#   VERSION       provider version to install (default 0.1.0); pass as arg to override
#   OS_ARCH       explicit target, e.g. linux_amd64 (default: auto-detect)
set -euo pipefail

VERSION="${1:-${VERSION:-0.1.0}}"
MIRROR_URL="${MIRROR_URL:-https://jitguard.com/terraform/}"
MIRROR_URL="${MIRROR_URL%/}"

HOSTNAME="registry.terraform.io"
NAMESPACE="barneyparker"
TYPE="jitguard"
BASE="$MIRROR_URL/$HOSTNAME/$NAMESPACE/$TYPE"

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
os="${OS_ARCH%_*}"
arch="${OS_ARCH#*_}"

echo "==> Querying mirror for provider v${VERSION} (${OS_ARCH})"
echo "    ${BASE}/index.json"

INDEX="$(curl -fsSL "$BASE/index.json")"
if ! echo "$INDEX" | grep -q "\"${VERSION}\""; then
  echo "error: version ${VERSION} not in mirror index:" >&2
  echo "$INDEX" >&2
  exit 1
fi

VERSION_JSON="$(curl -fsSL "$BASE/${VERSION}.json")"

# Extract the archive URL + first h1 hash for this platform.
ZIP_URL="$(echo "$VERSION_JSON" | python3 -c "
import json, sys
d = json.load(sys.stdin)
a = d.get('archives', {}).get('${OS_ARCH}')
if not a:
    sys.exit('error: no package for ${OS_ARCH} in mirror')
print(a['url'])
")"
H1="$(echo "$VERSION_JSON" | python3 -c "
import json, sys
d = json.load(sys.stdin)
a = d.get('archives', {}).get('${OS_ARCH}', {})
print(next((h for h in a.get('hashes', []) if h.startswith('h1:')), ''))
")"

if [ -z "$ZIP_URL" ]; then
  echo "error: no archive URL for ${OS_ARCH} (version ${VERSION} may not ship this platform)" >&2
  exit 1
fi
if [ -z "$H1" ]; then
  echo "error: mirror has no h1 hash for ${OS_ARCH} — refusing unverified download" >&2
  exit 1
fi

# URL is relative to the version manifest → resolves alongside it.
ZIP_BASE="$(dirname "$BASE/${VERSION}.json")"
DEST_DIR="${XDG_CONFIG_HOME:-$HOME/.terraform.d}/plugins/$HOSTNAME/$NAMESPACE/$TYPE/$VERSION/$OS_ARCH"
PLUGIN="$DEST_DIR/terraform-provider-jitguard_v$VERSION"

echo "==> Downloading ${ZIP_URL}"
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT
curl -fsSL -o "$TMPDIR/plugin.zip" "$ZIP_BASE/$ZIP_URL"

# Verify h1: base64(sha256(zip)).
ACTUAL_H1="h1:$(openssl dgst -sha256 -binary "$TMPDIR/plugin.zip" | openssl base64 -A | tr -d '\n')"
if [ "$ACTUAL_H1" != "$H1" ]; then
  echo "error: hash mismatch for $(basename "$ZIP_URL")" >&2
  echo "  expected: $H1" >&2
  echo "  actual:   $ACTUAL_H1" >&2
  exit 1
fi
echo "    verified h1 hash"

echo "==> Installing to ${DEST_DIR}"
mkdir -p "$DEST_DIR"
unzip -oq "$TMPDIR/plugin.zip" -d "$TMPDIR/unpacked"
cp "$TMPDIR/unpacked/terraform-provider-jitguard" "$PLUGIN"
chmod +x "$PLUGIN"

echo "==> Installed: ${PLUGIN}"
echo "    terraform init will now find it automatically (implicit filesystem mirror)."
