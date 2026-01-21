#!/bin/bash
set -euo pipefail

# Run golangci-lint
golangci-lint run --timeout=5m

