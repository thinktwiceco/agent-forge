#!/bin/bash
set -e

# Define the output directory
OUTPUT_DIR="python-sdk/bin"
mkdir -p "$OUTPUT_DIR"

echo "Building Python SDK binaries..."

# Linux amd64
echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -o "$OUTPUT_DIR/agentforge-server-linux-amd64" ./cmd/server/main.go

# macOS arm64 (Apple Silicon)
echo "Building for macOS (arm64)..."
GOOS=darwin GOARCH=arm64 go build -o "$OUTPUT_DIR/agentforge-server-darwin-arm64" ./cmd/server/main.go

# macOS amd64 (Intel)
echo "Building for macOS (amd64)..."
GOOS=darwin GOARCH=amd64 go build -o "$OUTPUT_DIR/agentforge-server-darwin-amd64" ./cmd/server/main.go

# Windows amd64
echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -o "$OUTPUT_DIR/agentforge-server-windows-amd64.exe" ./cmd/server/main.go

echo "Build complete! Binaries are located in $OUTPUT_DIR"
