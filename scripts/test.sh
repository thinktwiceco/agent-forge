#!/bin/bash
set -euo pipefail

# Parse arguments
# Parse arguments
UNIT_TESTS=false
INTEGRATION_TESTS=false

while [[ $# -gt 0 ]]; do
  case $1 in
    --unit)
      UNIT_TESTS=true
      shift
      ;;
    --integration)
      INTEGRATION_TESTS=true
      shift
      ;;
    *)
      echo "Unknown option: $1"
      echo "Usage: $0 [--unit] [--integration]"
      exit 1
      ;;
  esac
done

# Check if at least one test type is specified
if [ "$UNIT_TESTS" = false ] && [ "$INTEGRATION_TESTS" = false ]; then
  echo "No test type specified. Use --unit for unit tests and/or --integration for integration tests."
  exit 1
fi

# Run unit tests
if [ "$UNIT_TESTS" = true ]; then
  echo "Running Unit Tests..."
  go test -v -race -coverprofile=coverage.out ./src/... ./cmd/...
  echo ""
  echo "Total Coverage:"
  go tool cover -func=coverage.out | grep total: | awk '{print $3}'
fi

# Run integration tests
if [ "$INTEGRATION_TESTS" = true ]; then
  echo "Running Integration Tests..."
  go test -v -tags=integration ./src/agents/...
fi

