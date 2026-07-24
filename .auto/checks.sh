#!/bin/bash
# .auto/checks.sh — correctness checks for autoresearch
# Runs BEFORE log_experiment. If this fails, result is 'checks_failed'.
set -euo pipefail

echo "=== Checks ==="

# 1. go vet (fast, catches real bugs)
echo "--- go vet ---"
go vet ./... 2>&1

# 2. golangci-lint (if available)
if command -v golangci-lint >/dev/null 2>&1; then
    echo "--- golangci-lint ---"
    golangci-lint run ./... 2>&1
fi

# 3. Verify tests compile
echo "--- test compile check ---"
go test -c -o /dev/null ./... 2>&1

echo "=== All checks passed ==="
