#!/bin/bash
set -euo pipefail

# Ensure golangci-lint v2 is in PATH (prepend to prioritize it)
export PATH=$(go env GOPATH)/bin:$PATH

# Run golangci-lint
golangci-lint run --timeout=5m

