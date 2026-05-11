// Runtime config placeholder for dev. In production, deployments/docker-entrypoint.sh
// overwrites this with values from env vars. lib/env.ts falls back to Vite-injected
// VITE_* values when the runtime fields are missing.
window.__APP_CONFIG__ = {};
