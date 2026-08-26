#!/bin/sh
set -eu

mkdir -p ./data

sqlite3 ./data/todos.db < ./init.sql

docker compose up -d