#!/bin/bash
set -euo pipefail

# Start the agent-forge app in dev mode
# Run from project root; uses DEBUG logging and config from cmd/localforge

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT/cmd/localforge"

CONFIG="${1:-config.yaml}"
if [ ! -f "$CONFIG" ]; then
  CONFIG="config.example.yaml"
  if [ ! -f "$CONFIG" ]; then
    echo "No config found. Create cmd/localforge/config.yaml or config.example.yaml"
    exit 1
  fi
  echo "Using $CONFIG (config.yaml not found)"
fi

echo "Starting Local Forge (config: $CONFIG, port: 8080)..."
AF_LOG_LEVEL=DEBUG go run "$PROJECT_ROOT/cmd/localforge/src" -config "$CONFIG" -port 8080 -dev
