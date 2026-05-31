#!/bin/sh
set -e

goose -dir "${MIGRATIONS_DIR:-/app/migrations}" postgres "$DATABASE_URL" up

exec /bin/smartbid
