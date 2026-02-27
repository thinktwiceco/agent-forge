#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

AGENT_NAME="${1:-}"
if [ -z "$AGENT_NAME" ]; then
  echo "Usage: $0 <agent_name>"
  exit 1
fi

INSTALL_DIR="$PROJECT_ROOT/$AGENT_NAME"
BIN_DIR="$INSTALL_DIR/bin"

if [ ! -d "$INSTALL_DIR" ] || [ ! -f "$INSTALL_DIR/config.yaml" ]; then
  echo "Agent '$AGENT_NAME' not found. Run install.sh first."
  exit 1
fi

mkdir -p "$BIN_DIR"

echo "Rebuilding binary for $AGENT_NAME..."
cd "$PROJECT_ROOT"
go build -o "$BIN_DIR/localforge" ./cmd/localforge/src

echo "Update complete."
