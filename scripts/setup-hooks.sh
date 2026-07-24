#!/bin/sh
set -e

HOOK_SRC="scripts/pre-commit"
HOOK_DST=".git/hooks/pre-commit"

if [ -f "$HOOK_DST" ]; then
    echo "⚠ Pre-commit hook already exists at $HOOK_DST"
    echo "  Delete it first or run: rm $HOOK_DST"
    exit 1
fi

cp "$HOOK_SRC" "$HOOK_DST"
chmod +x "$HOOK_DST"
echo "✅ Pre-commit hook installed at $HOOK_DST"
