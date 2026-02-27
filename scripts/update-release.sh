#!/bin/bash
# update-release.sh — download and update the localforge binary to the latest version
# This script is meant to be run from within an agent installation directory.
#
# Usage:
#   ./update-release.sh
#
# It will detect the current OS/arch and update the binary in the bin/ folder.
set -euo pipefail

REPO="thinktwiceco/agent-forge"
API_URL="https://api.github.com/repos/${REPO}/releases/latest"

# ─── helpers ────────────────────────────────────────────────────────────────

die() { echo "ERROR: $*" >&2; exit 1; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "'$1' is required but not found. Please install it and retry."
}

# ─── dependency check ───────────────────────────────────────────────────────

require_cmd curl

# jq is optional — fall back to python3 / python if absent
if command -v jq >/dev/null 2>&1; then
  JSON_TOOL="jq"
elif command -v python3 >/dev/null 2>&1; then
  JSON_TOOL="python3"
elif command -v python >/dev/null 2>&1; then
  JSON_TOOL="python"
else
  die "Neither 'jq', 'python3', nor 'python' found. Install one of them and retry."
fi

json_field() {
  # usage: json_field <field> <json-string>
  local field="$1"
  local json="$2"
  case "$JSON_TOOL" in
    jq)     echo "$json" | jq -r ".$field" ;;
    python3) echo "$json" | python3 -c "import sys,json; print(json.load(sys.stdin)['$field'])" ;;
    python)  echo "$json" | python  -c "import sys,json; print(json.load(sys.stdin)['$field'])" ;;
  esac
}

json_asset_url() {
  # usage: json_asset_url <asset-name> <json-string>
  local name="$1"
  local json="$2"
  case "$JSON_TOOL" in
    jq)
      echo "$json" | jq -r --arg n "$name" '.assets[] | select(.name == $n) | .browser_download_url'
      ;;
    python3|python)
      echo "$json" | "$JSON_TOOL" -c "
import sys, json
name = '$name'
data = json.load(sys.stdin)
for a in data.get('assets', []):
    if a['name'] == name:
        print(a['browser_download_url'])
        sys.exit(0)
"
      ;;
  esac
}

# ─── OS / arch detection ─────────────────────────────────────────────────────

RAW_OS="$(uname -s 2>/dev/null || echo "unknown")"
RAW_ARCH="$(uname -m 2>/dev/null || echo "unknown")"

case "$RAW_OS" in
  Darwin)                     OS="darwin" ;;
  Linux)                      OS="linux"  ;;
  MINGW*|MSYS*|CYGWIN*|Windows_NT) OS="windows" ;;
  *)                          die "Unsupported OS: $RAW_OS" ;;
esac

case "$RAW_ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)             die "Unsupported architecture: $RAW_ARCH" ;;
esac

# Windows only ships amd64
if [ "$OS" = "windows" ] && [ "$ARCH" != "amd64" ]; then
  die "Windows binaries are only available for amd64 (detected: $ARCH)"
fi

EXT=""
[ "$OS" = "windows" ] && EXT=".exe"

echo "Detected OS: $OS / arch: $ARCH"

# ─── find installation directory ─────────────────────────────────────────────

# If running from ./update-release.sh, the script is in the root of the install dir
SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"
INSTALL_DIR="$SCRIPT_DIR"
BIN_DIR="$INSTALL_DIR/bin"
BINARY_PATH="$BIN_DIR/localforge${EXT}"

if [ ! -d "$BIN_DIR" ]; then
  die "bin/ directory not found at $BIN_DIR. Are you running this from an agent installation directory?"
fi

# ─── fetch latest release info ───────────────────────────────────────────────

echo "Fetching latest release from GitHub..."
RELEASE_JSON="$(curl -fsSL "$API_URL")" || die "Failed to fetch release info from $API_URL"

TAG="$(json_field tag_name "$RELEASE_JSON")"
[ -z "$TAG" ] && die "No releases found in repository $REPO"

VERSION="${TAG#v}"
ASSET_NAME="localforge-${VERSION}-${OS}-${ARCH}${EXT}"
DOWNLOAD_URL="$(json_asset_url "$ASSET_NAME" "$RELEASE_JSON")"
[ -z "$DOWNLOAD_URL" ] && die "No binary found for OS=$OS ARCH=$ARCH in release $TAG (looked for: $ASSET_NAME)"

echo "Latest release : $TAG"
echo "Binary         : $ASSET_NAME"

# ─── download and replace binary ──────────────────────────────────────────────

echo "Downloading $ASSET_NAME..."
curl -fSL --progress-bar -o "$BINARY_PATH" "$DOWNLOAD_URL"
chmod +x "$BINARY_PATH"

# ─── done ────────────────────────────────────────────────────────────────────

echo ""
echo "Update complete ($TAG):"
echo "  $BINARY_PATH"
echo ""
echo "You can now run ./start.sh (or start.bat on Windows) to use the updated agent."
