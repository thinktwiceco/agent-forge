#!/bin/bash
set -euo pipefail

# Parse arguments
UNIT_TESTS=false

while [[ $# -gt 0 ]]; do
  case $1 in
    --unit)
      UNIT_TESTS=true
      shift
      ;;
    *)
      echo "Unknown option: $1"
      echo "Usage: $0 [--unit]"
      exit 1
      ;;
  esac
done

# Run unit tests
if [ "$UNIT_TESTS" = true ]; then
  go test -v -race -coverprofile=coverage.out ./src/... ./cmd/...
else
  echo "No test type specified. Use --unit to run unit tests."
  exit 1
fi

