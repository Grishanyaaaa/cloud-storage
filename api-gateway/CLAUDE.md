# CLAUDE.md — API Gateway

## Project Overview

This is a Go **1.26.2** API Gateway microservice. All code must target Go 1.26.2 and leverage its standard library features where appropriate.

The API Gateway serves as the single entry point for all client requests, handling:
- Request routing to upstream services
- JWT token validation for protected endpoints
- Rate limiting
- CORS handling
- Health checks

---

## Architecture

The gateway follows a simplified 3-layer architecture:

```
presentation  →  infrastructure
```

- **`infrastructure`** — JWKS client for JWT validation, configuration loading, HTTP clients for upstream services
- **`presentation`** — HTTP handlers (chi), middleware (JWT auth, CORS, rate limiting), reverse proxy logic

No domain or application layer is needed since the gateway doesn't contain business logic — it only routes and validates.

---

## Project Structure

```
.
├── cmd/
│   └── api-gateway/
│       └── main.go          # wires all layers, zero business logic
├── internal/
│   ├── infrastructure/
│   │   ├── config/          # configuration structs + env loading
│   │   └── client/          # JWKS client for JWT validation
│   └── presentation/
│       └── http/
│           ├── handler/     # proxy handler, health checks
│           ├── middleware/  # JWT auth, CORS, rate limiting
│           ├── router.go    # route registration
│           └── server.go    # HTTP server wrapper
├── pkg/                     # shared packages (logger)
├── deployments/
│   └── k8s/                 # Kubernetes manifests
└── Dockerfile
```

---

## Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go) and the [Google Go Style Guide](https://google.github.io/styleguide/go/).
- **Formatting**: all code must pass `gofmt` and `goimports`. Never submit unformatted code.
- **Line length**: soft limit 100 characters; hard limit 120.
- **Package names**: short, lowercase, no underscores (`middleware`, not `middle_ware`).
- **Error variables**: prefix with `Err` (`ErrInvalidToken`, `ErrUpstreamUnavailable`).
- **Comments**: all exported identifiers must have a godoc comment starting with the identifier name.

---

## Routing Strategy

### Public Routes (no JWT required)
- `/auth/v1/*` — proxied to auth-service (register, login, refresh, logout)
- `/.well-known/jwks.json` — proxied to auth-service
- `/health` — gateway health check
- `/ready` — gateway + upstream readiness check

### Protected Routes (JWT required)
- `/api/*` — future protected endpoints, JWT validation via middleware

---

## JWT Validation

The gateway validates JWT tokens using JWKS fetched from auth-service:

1. **Startup**: fetch JWKS from `http://auth-service/.well-known/jwks.json`
2. **Background refresh**: refresh JWKS every 1 hour (configurable)
3. **Request validation**: extract token from `Authorization: Bearer <token>`, validate signature using cached public keys
4. **Context injection**: add `user_id` and `email` to request context for downstream handlers

**Algorithm**: Ed25519 (EdDSA)  
**Issuer**: `auth-service`  
**Audience**: `cloud-storage`

---

## Reverse Proxy

The gateway uses a custom reverse proxy handler that:
- Preserves original request path and query parameters
- Copies headers (excluding hop-by-hop headers)
- Sets `X-Forwarded-For`, `X-Forwarded-Proto`, `X-Forwarded-Host`
- Streams response back to client
- Returns 502 Bad Gateway if upstream is unavailable

**Timeout**: 30 seconds per upstream request

---

## Rate Limiting

Per-IP rate limiting using `golang.org/x/time/rate`:
- **Rate**: 100 requests/second
- **Burst**: 200 requests
- **Cleanup**: old limiters removed every 5 minutes

Returns `429 Too Many Requests` when limit exceeded.

---

## Health Checks

### `/health`
Always returns `200 OK` if gateway is running. Used for liveness probe.

### `/ready`
Checks gateway + upstream services:
- Queries `http://auth-service/.well-known/jwks.json`
- Returns `200 OK` if all services healthy
- Returns `503 Service Unavailable` if any service unhealthy

Used for readiness probe.

---

## Configuration

All configuration comes from environment variables (12-factor):

```bash
# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_READ_TIMEOUT=10s
SERVER_WRITE_TIMEOUT=15s
SERVER_IDLE_TIMEOUT=60s
SERVER_SHUTDOWN_TIMEOUT=30s

# Upstream services
AUTH_SERVICE_URL=http://auth-service

# JWT validation
JWT_JWKS_URL=http://auth-service/.well-known/jwks.json
JWT_JWKS_REFRESH_INTERVAL=1h
JWT_ISSUER=auth-service
JWT_AUDIENCE=cloud-storage

# CORS
CORS_ALLOW_ORIGINS=*
CORS_ALLOW_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOW_HEADERS=Content-Type,Authorization
CORS_ALLOW_CREDENTIALS=false
CORS_MAX_AGE=86400
```

---

## Logging

- Logger: **`log/slog`** (stdlib). Do not introduce `zap`, `logrus`, or any other logger.
- Structured logging only — no `fmt.Println` or `log.Printf` in production code.
- Log levels: `DEBUG` for dev traces, `INFO` for lifecycle events, `WARN` for recoverable issues, `ERROR` for failures requiring attention.

---

## Error Handling

### Upstream errors
- **502 Bad Gateway**: upstream service unavailable or timeout
- **401 Unauthorized**: invalid or expired JWT token
- **429 Too Many Requests**: rate limit exceeded

### Wrapping and unwrapping
- Always wrap errors with context: `fmt.Errorf("fetch jwks: %w", err)`.
- Never discard errors with `_`. If an error is truly ignorable, add a comment explaining why.

---

## Concurrency

- Use `context.Context` as the **first parameter** in every function that does I/O or is long-running.
- Never store a context in a struct field.
- JWKS background refresh runs in a goroutine, cancelled via context on shutdown.

---

## HTTP (presentation layer)

- Router: **[chi](https://github.com/go-chi/chi)**. Do not introduce any other router.
- All routes are registered in `internal/presentation/http/router.go`.
- Middleware chain order:
  ```
  chimiddleware.RequestID
  chimiddleware.Logger
  chimiddleware.Recoverer
  chimiddleware.CleanPath
  CORS
  RateLimiter
  JWTAuth (protected routes only)
  ```
- Timeouts: every outbound HTTP call must carry a deadline via context or `http.Client.Timeout`.

---

## Deployment

### Docker
Multi-stage build using Go 1.26.2 alpine:
```bash
docker build -t api-gateway:latest .
```

### Kubernetes
Manifests in `deployments/k8s/`:
- `configmap.yaml` — environment configuration
- `deployment.yaml` — gateway deployment (1 replica)
- `service.yaml` — ClusterIP service
- `ingress.yaml` — Ingress resource for external access

**Ingress**: routes `cloud-storage.local` → api-gateway → auth-service

---

## CI / Quality Gates

The following must pass on every PR:

```
go build ./...
go test ./... -race -count=1
go vet ./...
staticcheck ./...
golangci-lint run
```

Do not merge code that fails any gate, even with a "will fix later" comment.

---

## What Claude Should NOT Do

- Do **not** refactor working code just to match a preferred style unless asked.
- Do **not** introduce new dependencies without discussing trade-offs first.
- Do **not** add business logic to the gateway — it belongs in upstream services.
- Do **not** generate `panic`-based error handling outside `main`.
- Do **not** add global mutable state.
- Do **not** silently truncate or swallow errors.
- Do **not** log inside middleware by importing `slog` globally — inject via constructor if needed.
