#!/bin/sh
# Generates /usr/share/nginx/html/config.js from env vars on container start.
# Read by index.html before the main bundle so window.__APP_CONFIG__ exists
# by the time lib/env.ts is evaluated.
set -eu

OUT="/usr/share/nginx/html/config.js"
cat > "$OUT" <<EOF
window.__APP_CONFIG__ = {
  API_BASE_URL: "${API_BASE_URL:-}",
  SHARE_BASE_URL: "${SHARE_BASE_URL:-}"
};
EOF
echo "[entrypoint] wrote $OUT"
