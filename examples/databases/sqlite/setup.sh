#!/bin/sh
set -eu

SCRIPT_DIR="$(dirname "$0")"

mkdir -p "$SCRIPT_DIR/data"
sqlite3 "$SCRIPT_DIR/data/app.db" < "$SCRIPT_DIR/init.sql"

echo "Initialized SQLite database at $SCRIPT_DIR/data/app.db"