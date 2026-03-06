#!/bin/bash
# install-release.sh — download the latest localforge binary from GitHub Releases
# and create the required folder structure for the application.
#
# Usage:
#   ./scripts/install-release.sh [install-dir]
#
# If install-dir is omitted, you will be prompted for an agent name and the
# directory will be created as ./<agent-name> relative to your current working directory.
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

# ─── install directory ───────────────────────────────────────────────────────

if [ "${1:-}" != "" ]; then
  INSTALL_DIR="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
  AGENT_NAME="$(basename "$INSTALL_DIR")"
else
  # If stdin is not a terminal (e.g., piped), re-open from /dev/tty if available
  # Otherwise use a default agent name
  if [ -t 0 ] || [ -c /dev/tty ]; then
    echo ""
    read -r -p "Agent name (will create ./<name>/ directory): " AGENT_NAME < /dev/tty 2>/dev/null || AGENT_NAME=""
  fi
  
  if [ -z "$AGENT_NAME" ]; then
    # Use environment variable or default
    AGENT_NAME="${AGENT_FORGE_NAME:-my-agent}"
  fi
  
  INSTALL_DIR="$(pwd)/$AGENT_NAME"
fi

BIN_DIR="$INSTALL_DIR/bin"
BINARY_PATH="$BIN_DIR/localforge${EXT}"

echo ""
echo "Install directory: $INSTALL_DIR"

if [ -d "$INSTALL_DIR" ]; then
  echo "Directory already exists."
fi

# ─── create folder structure ─────────────────────────────────────────────────

mkdir -p "$BIN_DIR"
mkdir -p "$INSTALL_DIR/procedures"
mkdir -p "$INSTALL_DIR/data"

echo "Created folder structure."

# ─── download binary ─────────────────────────────────────────────────────────

echo "Downloading $ASSET_NAME..."
curl -fSL --progress-bar -o "$BINARY_PATH" "$DOWNLOAD_URL"
chmod +x "$BINARY_PATH"
echo "Binary installed: $BINARY_PATH"

# ─── config.yaml ─────────────────────────────────────────────────────────────

if [ -f "$INSTALL_DIR/config.yaml" ]; then
  echo "config.yaml already exists — skipping (delete it to regenerate)."
else
  # Try to use external template file, fallback to embedded heredoc
  TEMPLATE_FILE="$(dirname "$0")/../installation/templates/config.yaml.template"
  if [ -f "$TEMPLATE_FILE" ]; then
    # Replace $AGENT_NAME but keep ${AGENT_WORKING_DIR} as literal
    sed "s/\$AGENT_NAME/$AGENT_NAME/g" "$TEMPLATE_FILE" > "$INSTALL_DIR/config.yaml"
    echo "config.yaml created."
  else
    cat > "$INSTALL_DIR/config.yaml" << EOF
# Minimal agent configuration - customize as needed
agent:
  name: "$AGENT_NAME"
  system_prompt: |
    You are a helpful assistant. Customize this prompt for your use case.
  model: "togetherai::moonshotai/Kimi-K2.5"
  working_dir: "\${AGENT_WORKING_DIR}"
  persistence: "json"
  tools:
    - name: fs
    - name: git
  subagents:
    reasoning: "deepseek::deepseek-reasoner"
EOF
    echo "config.yaml created."
  fi
fi

# ─── .env placeholder ────────────────────────────────────────────────────────

if [ ! -f "$INSTALL_DIR/.env" ]; then
  # Try to use external template file, fallback to embedded heredoc
  TEMPLATE_FILE="$(dirname "$0")/../installation/templates/.env.template"
  if [ -f "$TEMPLATE_FILE" ]; then
    # Replace $AGENT_WORKING_DIR with actual install directory
    sed "s|\$AGENT_WORKING_DIR|$INSTALL_DIR|g" "$TEMPLATE_FILE" > "$INSTALL_DIR/.env"
    echo ".env placeholder created."
  else
    cat > "$INSTALL_DIR/.env" << ENVEOF
# Add your API keys here. This file is NOT committed to git.
AGENT_WORKING_DIR=$INSTALL_DIR
# AF_TOGETHERAI_API_KEY=your_key_here
# AF_OPENAI_API_KEY=your_key_here
# AF_DEEPSEEK_API_KEY=your_key_here
ENVEOF
    echo ".env placeholder created."
  fi
fi

# ─── launcher ────────────────────────────────────────────────────────────────

if [ "$OS" = "windows" ]; then
  LAUNCHER="$INSTALL_DIR/start.bat"
  if [ ! -f "$LAUNCHER" ]; then
    # Try to use external template file, fallback to embedded heredoc
    TEMPLATE_FILE="$(dirname "$0")/../installation/templates/start.bat.template"
    if [ -f "$TEMPLATE_FILE" ]; then
      cp "$TEMPLATE_FILE" "$LAUNCHER"
      echo "start.bat created."
    else
      cat > "$LAUNCHER" << 'BATEOF'
@echo off
SET AGENT_WORKING_DIR=%~dp0
cd /d %~dp0
bin\localforge.exe -config config.yaml -port 8080
BATEOF
      echo "start.bat created."
    fi
  else
    echo "start.bat already exists — skipping."
  fi
else
  LAUNCHER="$INSTALL_DIR/start.sh"
  if [ ! -f "$LAUNCHER" ]; then
    # Try to use external template file, fallback to embedded heredoc
    TEMPLATE_FILE="$(dirname "$0")/../installation/templates/start.sh.template"
    if [ -f "$TEMPLATE_FILE" ]; then
      cp "$TEMPLATE_FILE" "$LAUNCHER"
      chmod +x "$LAUNCHER"
      echo "start.sh created."
    else
      cat > "$LAUNCHER" << 'STARTEOF'
#!/bin/bash
set -euo pipefail
SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"
INSTALL_DIR="$SCRIPT_DIR"
export AGENT_WORKING_DIR="$INSTALL_DIR"
cd "$INSTALL_DIR"
exec ./bin/localforge -config config.yaml -port 8080
STARTEOF
      chmod +x "$LAUNCHER"
      echo "start.sh created."
    fi
  else
    echo "start.sh already exists — skipping."
  fi
fi

# ─── copy update script ───────────────────────────────────────────────────────

UPDATE_SCRIPT="$INSTALL_DIR/update-release.sh"

# Attempt to locate update-release.sh script
# When piped (curl | bash), we fetch it from GitHub; when run locally, copy from scripts/
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "https://raw.githubusercontent.com/thinktwiceco/agent-forge/main/scripts/update-release.sh" -o "$UPDATE_SCRIPT" 2>/dev/null && chmod +x "$UPDATE_SCRIPT" && echo "update-release.sh downloaded." || echo "WARNING: Could not download update-release.sh (skipped)"
else
  echo "WARNING: curl not available to download update-release.sh (skipped)"
fi

# ─── done ────────────────────────────────────────────────────────────────────

echo ""
echo "Install complete ($TAG):"
echo "  $BINARY_PATH"
echo "  $INSTALL_DIR/config.yaml  — edit model, system_prompt, tools"
echo "  $INSTALL_DIR/.env         — add your API keys"
echo "  $UPDATE_SCRIPT            — run to update to latest version"
echo ""
if [ "$OS" = "windows" ]; then
  echo "Start: cd \"$INSTALL_DIR\" && start.bat"
else
  echo "Start: cd \"$INSTALL_DIR\" && ./start.sh"
fi
echo ""
echo "Update: cd \"$INSTALL_DIR\" && ./update-release.sh"
