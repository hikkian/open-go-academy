#!/bin/sh
set -eu
e2e_db=${ACADEMY_DB:-.data/e2e/academy.db}
e2e_dir=$(dirname "$e2e_db")
mkdir -p "$e2e_dir"
if [ -z "${ACADEMY_DB:-}" ]; then
  rm -f "$e2e_db" "$e2e_db-shm" "$e2e_db-wal" "$e2e_dir/session.secret" "$e2e_dir/issuer.key"
fi
ACADEMY_ADDR=${ACADEMY_ADDR:-127.0.0.1:8081} ACADEMY_DB="$e2e_db" exec sh scripts/go.sh run ./cmd/api
