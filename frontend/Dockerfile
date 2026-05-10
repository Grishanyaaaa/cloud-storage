# syntax=docker/dockerfile:1.7
# Multi-stage build: node-builder → nginx-runtime
# - Build args: VITE_API_BASE_URL (defaulted at build time, also overridable
#   at runtime via window.__APP_CONFIG__ injected by the entrypoint script).

FROM node:20-alpine AS deps
WORKDIR /app
ENV CI=1
COPY package.json package-lock.json* ./
RUN --mount=type=cache,target=/root/.npm npm ci

FROM node:20-alpine AS build
WORKDIR /app
ENV NODE_ENV=production
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Compile the route tree, type-check, and produce dist/
ARG VITE_API_BASE_URL=/
ARG VITE_SHARE_BASE_URL=
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_SHARE_BASE_URL=$VITE_SHARE_BASE_URL
RUN npm run build

# ----------------------------------------------------------------------
FROM nginx:1.27-alpine AS runtime

# nginx.conf with SPA fallback + tight CSP + immutable static caching.
COPY deployments/nginx.conf /etc/nginx/conf.d/default.conf

# Built static assets land at /usr/share/nginx/html.
COPY --from=build /app/dist /usr/share/nginx/html

# Runtime config injection: writes a tiny config.js based on env vars at
# container start, then exec's nginx. This means a single image can be
# deployed against staging / prod without rebuild.
COPY deployments/docker-entrypoint.sh /docker-entrypoint.d/40-runtime-config.sh
RUN chmod +x /docker-entrypoint.d/40-runtime-config.sh

EXPOSE 80
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
  CMD wget -q -O - http://127.0.0.1/healthz >/dev/null || exit 1
