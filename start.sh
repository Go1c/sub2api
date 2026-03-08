#!/bin/sh
set -e

echo "Running database migrations..."
PRISMA_BIN=$(find node_modules/.pnpm -path '*/prisma/build/index.js' -type f | head -1)

if [ -n "$PRISMA_BIN" ]; then
  node "$PRISMA_BIN" migrate deploy --config prisma.config.ts
  echo "Migrations complete."
else
  echo "Warning: prisma CLI not found, skipping migrations."
fi

echo "Starting application..."
exec node server.js
