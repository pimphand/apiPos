#!/bin/sh
set -e

# Run database migrations before starting the application
if [ -n "$DATABASE_HOST" ]; then
  echo "Running database migrations..."
  migrate -path /app/db/migration \
    -database "postgres://${DATABASE_USER}:${DATABASE_PASSWORD}@${DATABASE_HOST}:${DATABASE_PORT}/${DATABASE_NAME}?sslmode=disable" \
    up || echo "Migrations already applied or failed; continuing startup"
fi

exec ./main
