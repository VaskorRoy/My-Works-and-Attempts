#!/usr/bin/env bash
# Launch the Invoice Tracker application.
# Unset cross-compilation environment variables that may interfere.
unset CC CXX

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BINARY="$SCRIPT_DIR/build/bin/invoice-app"

if [ ! -f "$BINARY" ]; then
    echo "Error: Binary not found at $BINARY"
    echo "Run 'wails build -tags webkit2_41' first."
    exit 1
fi

chmod +x "$BINARY"
exec "$BINARY" "$@"
