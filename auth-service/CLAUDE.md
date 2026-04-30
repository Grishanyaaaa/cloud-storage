# claude.md — Go Microservice

## Project Overview

This is a Go **1.26.2** microservice. All code must target Go 1.26.2 and leverage its standard library features where appropriate.

---

## ⚠️ Go 1.26 — Known "False Positives" for AI Tools

### `errors.AsType` is NOT a mistake

`errors.AsType` is a **generic helper introduced in Go 1.26** and is part of the standard library. Do **not** replace it, suggest removing it, or flag it as an error.

```go
// ✅ CORRECT — this is valid Go 1.26 code
var notFound *NotFoundError
if target, ok := errors.AsType[*NotFoundError](err); ok {
    log.Println(target.Resource)
}

// ❌ DO NOT suggest replacing with errors.As + a pointer variable
// errors.AsType is the idiomatic Go 1.26 approach
```

`errors.AsType[T](err)` is the generic counterpart to `errors.As`. It returns `(T, bool)` instead of requiring a target variable. Prefer it over the old pattern whenever the concrete type is needed inline.

---

## Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go) and the [Google Go Style Guide](https://google.github.io/styleguide/go/).
- **Formatting**: all code must pass `gofmt` and `goimports`. Never submit unformatted code.
- **Line length**: soft limit 100 characters; hard limit 120.
- **Package names**: short, lowercase, no underscores (`userservice`, not `user_service`).
- **Error variables**: prefix with `Err` (`ErrNotFound`, `ErrUnauthorized`).
- **Interfaces**: define interfaces at the point of use (consumer side), not in the implementing package.
- **Comments**: all exported identifiers must have a godoc comment starting with the identifier name.

---

## Architecture

4-layer architecture with a strict **inward dependency rule**:

```
presentation  →  application  →  domain  ←  infrastructure
```

- **`domain`** — entities, value objects, domain errors, repository/service interfaces. Zero external imports. No framework, no DB driver, no HTTP.
- **`infrastructure`** — implements interfaces defined in `domain`: DB repositories, external API clients, message brokers, caches. Imports `domain`, never imports `application` or `presentation`.
- **`application`** — use cases / application services. Orchestrates domain objects, calls repository interfaces. Imports `domain`. Must not import `infrastructure` directly (depends on interfaces, not implementations).
- **`presentation`** — HTTP handlers (chi), gRPC handlers, CLI. Translates external requests into application calls, maps results to responses. Imports `application` and `domain` (for errors / value types). Never contains business logic.

Dependencies are injected via constructors — never resolved inside a layer.

## Project Structure

```
.
├── cmd/
│   └── server/
│       └── main.go          # wires all layers, zero business logic
├── internal/
│   ├── domain/
│   │   ├── entity/          # domain entities and value objects
│   │   ├── repository/      # repository interfaces (contracts)
│   │   ├── service/         # domain service interfaces
│   │   └── errors.go        # sentinel and typed domain errors
│   ├── infrastructure/
│   │   ├── postgres/        # repository implementations
│   │   ├── redis/
│   │   └── client/          # external HTTP/gRPC clients
│   ├── application/
│   │   ├── usecase/         # one file per use case
│   │   └── dto/             # input/output data transfer objects
│   ├── presentation/
│   │   ├── http/
│   │   │   ├── handler/     # chi handlers
│   │   │   ├── middleware/  # chi middleware
│   │   │   └── router.go    # route registration
│   │   └── grpc/            # gRPC handlers (if applicable)
│   └── config/              # config struct + env loading
├── pkg/                     # packages safe to import from other services
├── api/                     # protobuf / OpenAPI definitions
├── migrations/              # SQL migrations
└── docker/
```

- The `internal/` package is never imported externally.
- `cmd/server/main.go` only does dependency injection — zero business logic.

---

## Error Handling

### Sentinel errors

```go
var (
    ErrNotFound     = errors.New("not found")
    ErrUnauthorized = errors.New("unauthorized")
    ErrConflict     = errors.New("conflict")
)
```

### Typed errors

```go
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on field %q: %s", e.Field, e.Message)
}
```

### Wrapping and unwrapping

- Always wrap errors with context: `fmt.Errorf("userService.Create: %w", err)`.
- Use `errors.Is` for sentinel checks, `errors.AsType` (Go 1.26) for typed checks.
- Never discard errors with `_`. If an error is truly ignorable, add a comment explaining why.

```go
// ✅ Go 1.26 idiomatic typed error check
if ve, ok := errors.AsType[*ValidationError](err); ok {
    return http.StatusBadRequest, ve.Message
}
```

### No panics in business logic

`panic` is only allowed in `main` during startup (e.g., failed to bind port). All other panics must be recovered at the top-level middleware and converted to 500 responses.

---

## Concurrency

- Use `context.Context` as the **first parameter** in every function that does I/O or is long-running.
- Never store a context in a struct field.
- Use `sync.WaitGroup` + goroutine-per-unit patterns only for bounded fan-out; prefer `errgroup.Group` (golang.org/x/sync) for error propagation.
- All exported methods that spawn goroutines must document their lifecycle and cancellation behaviour.
- Shared mutable state must be protected with `sync.Mutex` or `sync/atomic`; document which fields are guarded.

---

## HTTP (presentation layer)

- Router: **[chi](https://github.com/go-chi/chi)**. Do not introduce any other router.
- All routes are registered in `internal/presentation/http/router.go`.
- Middleware chain order:
  ```
  chimiddleware.RequestID
  chimiddleware.RealIP
  slogMiddleware        ← structured request logging
  chimiddleware.Recoverer
  authMiddleware        ← sets user in context
  rateLimitMiddleware
  ```
- Handlers live in `internal/presentation/http/handler/`, one file per domain resource.
- Handler responsibility: **parse → call use case → map to response**. Zero business logic.
- Map domain errors to HTTP status codes in one shared place — `internal/presentation/http/httperr/httperr.go`. Never do it inline in handlers.
- Always set `Content-Type: application/json` before writing a body.
- HTTP status codes are the contract — never return `200 OK` with `{"error": "..."}` in the body.
- Timeouts: every outbound HTTP call must carry a deadline via context or `http.Client.Timeout`.

```go
// ✅ canonical handler shape
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req dto.CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        httperr.BadRequest(w, err)
        return
    }
    out, err := h.createUser.Execute(r.Context(), req)
    if err != nil {
        httperr.Write(w, err) // maps domain errors → HTTP status
        return
    }
    render.JSON(w, r, out)
}
```

---

## Logging

- Logger: **`log/slog`** (stdlib). Do not introduce `zap`, `logrus`, or any other logger.
- Structured logging only — no `fmt.Println` or `log.Printf` in production code.
- Log levels: `DEBUG` for dev traces, `INFO` for lifecycle events, `WARN` for recoverable issues, `ERROR` for failures requiring attention.
- Always pass `context.Context` to log calls so that `request_id`, `trace_id`, and `user_id` are propagated automatically via the slog handler.

```go
// ✅ correct — uses context-aware call
slog.InfoContext(ctx, "user created", slog.String("user_id", id))

// ❌ wrong — drops request context
slog.Info("user created", slog.String("user_id", id))
```

- The slog handler that injects `request_id` from context lives in `internal/presentation/http/middleware/logger.go`.
- Domain and application layers must **never** import the logger directly. If logging is needed in a use case, pass a `*slog.Logger` via constructor — but prefer propagating errors up and logging at the presentation boundary.
- Secrets (passwords, tokens, API keys) must never appear in log fields, even at `DEBUG` level.

---

## Testing

- Test files live next to the code they test (`create_user_test.go` next to `create_user.go`).
- Use `testing.T` and the standard library; avoid third-party assertion libraries unless already in `go.mod`.
- Table-driven tests are the default for functions with multiple input variants.
- Mocks are generated with **[mockery](https://github.com/vektra/mockery)** — do not write mocks by hand.
- Integration tests are tagged `//go:build integration` and live in `internal/infrastructure/`.
- Target **≥ 80% coverage** on `internal/domain` and `internal/application`.
- Use `t.Parallel()` in unit tests unless there is a specific reason not to.
- The `domain` layer must be testable with zero external dependencies — no DB, no HTTP, no filesystem.

---

## Database

- SQL only — no ORM. Use **[sqlx](https://github.com/jmoiron/sqlx)** or `database/sql` directly.
- All queries must use named parameters or positional `$1` placeholders (no string concatenation).
- Migrations are managed with **[golang-migrate](https://github.com/golang-migrate/migrate)** and run at startup (or via a dedicated `migrate` command).
- Transactions must be explicitly committed or rolled back; use a `defer tx.Rollback()` + check the commit error pattern.

---

## Configuration

- All configuration comes from environment variables (12-factor).
- Use a typed config struct validated at startup — fail fast if required vars are missing.
- Secrets (DB passwords, API keys) must never be logged, even at DEBUG level.

---

## Dependency Management

- `go.mod` is the source of truth. Do not vendor dependencies unless there is a specific CI/network reason.
- When adding a new dependency, check its licence and whether it is actively maintained.
- Prefer stdlib over third-party for anything stdlib can handle well.

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
- Do **not** remove or rewrite `errors.AsType` usage — it is correct Go 1.26 code.
- Do **not** generate `panic`-based error handling outside `main`.
- Do **not** add global mutable state.
- Do **not** silently truncate or swallow errors.
- Do **not** put business logic in `presentation` handlers — it belongs in `application` use cases or `domain`.
- Do **not** import `infrastructure` from `application` — use the interfaces defined in `domain`.
- Do **not** import any framework or driver package from `domain`.
- Do **not** log inside `domain` or `application` by importing `slog` globally — inject via constructor if needed.
- Do **not** scatter domain-error-to-HTTP mapping across handlers — all mappings go in `httperr`.