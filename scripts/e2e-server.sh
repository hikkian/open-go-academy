#!/bin/sh
set -eu
mkdir -p .data/e2e
rm -f .data/e2e/academy.db .data/e2e/academy.db-shm .data/e2e/academy.db-wal .data/e2e/session.secret
ACADEMY_ADDR=127.0.0.1:8081 ACADEMY_DB=.data/e2e/academy.db exec sh scripts/go.sh run ./cmd/api
