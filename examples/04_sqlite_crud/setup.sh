#!/bin/sh
set -eu

SCRIPT_DIR="$(dirname "$0")"

mkdir -p "$SCRIPT_DIR/data"
sqlite3 "$SCRIPT_DIR/data/todos.db" < "$SCRIPT_DIR/init.sql"

echo "Initialized $SCRIPT_DIR/data/todos.db"