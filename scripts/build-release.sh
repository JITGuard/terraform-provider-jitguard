#!/bin/bash
#
# Build registry-shaped provider artifacts locally with GoReleaser, without
# publishing. Normal releases are produced by .github/workflows/release.yml
# from a merge to main.
#
# Usage:
#   ./scripts/build-release.sh
#
# Output: dist/ containing the per-platform zips, the manifest, SHA256SUMS and
# its GPG signature (if GPG_FINGERPRINT and GPG_PASSPHRASE_FILE are set).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

exec goreleaser release --snapshot --clean
