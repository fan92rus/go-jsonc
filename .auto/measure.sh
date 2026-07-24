#!/bin/bash
# .auto/measure.sh — measure.sh for JSONC CST PBT test suite
set -euo pipefail

# Run tests and count passing/failing
# We count each individual rapid property check as a "test"
# The primary metric is: passing_tests (higher is better)
# Secondary: total_tests, pass_rate

output=$(go test -count=1 -timeout 60s ./... 2>&1) || true

# Count specific result types
passed=$(echo "$output" | grep -c "^--- PASS:" || true)
failed=$(echo "$output" | grep -c "^--- FAIL:" || true)
panics=$(echo "$output" | grep -c "^panic:" || true)

# Total distinct test functions
test_funcs=$(echo "$output" | grep "^=== RUN" | wc -l || true)

# Check for overall pass/fail
if echo "$output" | grep -q "^FAIL"; then
    overall="fail"
else
    overall="pass"
fi

# Report panics separately  
if [ "$panics" -gt 0 ]; then
    echo "WARNING: $panics panics detected" >&2
fi

echo "METRIC passing_tests=$passed"
echo "METRIC total_test_functions=$test_funcs"
echo "METRIC failed_tests=$failed"
echo "METRIC panics=$panics"

# Print failure summary for debugging
if [ "$failed" -gt 0 ] || [ "$panics" -gt 0 ]; then
    echo "--- FAILURE DETAILS ---" >&2
    echo "$output" | grep -E "^--- FAIL:|^panic:|^FAIL\s" | head -20 >&2
    echo "--- END FAILURES ---" >&2
fi
