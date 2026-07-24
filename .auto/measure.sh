#!/bin/bash
# .auto/measure.sh
set -euo pipefail

output=$(go test -count=1 -timeout 60s -v ./... 2>&1) || true

# Count individual test results
passed=$(echo "$output" | grep -c "^--- PASS:" || true)
failed=$(echo "$output" | grep -c "^--- FAIL:" || true)

# Count distinct test function names
test_funcs=$(echo "$output" | grep "^=== RUN" | wc -l || true)

# Panics
panics=$(echo "$output" | grep -c "^panic:" || true)

echo "METRIC passing_tests=$passed"
echo "METRIC failed_tests=$failed"
echo "METRIC total_test_functions=$test_funcs"
echo "METRIC panics=$panics"

# Failure details to stderr
if [ "$failed" -gt 0 ] || [ "$panics" -gt 0 ]; then
    echo "--- FAILURE SUMMARY ---" >&2
    echo "$output" | grep "^--- FAIL:" >&2
    [ "$panics" -gt 0 ] && echo "$output" | grep "^panic:" >&2
    echo "--- END ---" >&2
fi
