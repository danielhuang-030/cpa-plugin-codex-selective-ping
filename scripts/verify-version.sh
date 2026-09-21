#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
VERSION="$(sed -n 's/^VERSION=//p' Makefile | head -1)"
REG_VERSION="$(python3 -c 'import json; print(json.load(open("registry.json"))["plugins"][0]["version"])')"
if [[ "$REG_VERSION" != "$VERSION" ]]; then
  echo "registry.json version ($REG_VERSION) != Makefile VERSION ($VERSION)" >&2
  echo "CPA store shows registry version for uninstalled plugins; keep them in sync." >&2
  exit 1
fi
if ! grep -Eq "version[[:space:]]*=[[:space:]]*\"${VERSION}\"" registration.go; then
  echo "registration.go does not declare version \"$VERSION\"" >&2
  exit 1
fi
echo "version sync OK: $VERSION"
