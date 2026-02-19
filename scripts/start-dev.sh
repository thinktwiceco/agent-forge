#!/bin/bash
set -euo pipefail

# Start the agent-forge app in dev mode
# Run from project root; uses DEBUG logging and config from cmd/app

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

CONFIG="${1:-cmd/app/config.yaml}"
if [ ! -f "$CONFIG" ]; then
  CONFIG="cmd/app/config.example.yaml"
  if [ ! -f "$CONFIG" ]; then
    echo "No config found. Create cmd/app/config-test.yaml or config.example.yaml"
    exit 1
  fi
  echo "Using $CONFIG (config-test.yaml not found)"
fi

echo "Starting agent-forge (config: $CONFIG, port: 8080)..."
AF_LOG_LEVEL=DEBUG go run ./cmd/app -config "$CONFIG" -port 8080
