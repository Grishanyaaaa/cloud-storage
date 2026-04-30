# API Gateway

API Gateway для cloud-storage микросервисной архитектуры. Единая точка входа для всех клиентских запросов с поддержкой JWT-аутентификации, rate limiting и reverse proxy.

## Возможности

- **Reverse Proxy** — проксирование запросов к upstream сервисам (auth-service)
- **JWT Validation** — валидация токенов через JWKS от auth-service
- **Rate Limiting** — 100 req/s per IP с burst 200
- **CORS** — настраиваемые CORS headers
- **Health Checks** — `/health` и `/ready` endpoints для k8s probes
- **Graceful Shutdown** — корректное завершение при SIGTERM/SIGINT

## Архитектура

```
Ingress (nginx) → API Gateway → Auth Service
                              → Future Services
```

### Маршрутизация

**Публичные маршруты** (без JWT):
- `POST /auth/v1/register` → auth-service
- `POST /auth/v1/login` → auth-service
- `POST /auth/v1/refresh` → auth-service
- `POST /auth/v1/logout` → auth-service
- `GET /.well-known/jwks.json` → auth-service

**Защищённые маршруты** (требуют JWT):
- `/api/*` → будущие защищённые endpoints

**Health checks**:
- `GET /health` — liveness probe
- `GET /ready` — readiness probe (проверяет upstream)

## Быстрый старт

### Локальная разработка

```bash
# 1. Скопировать пример конфигурации
cp deployments/.env.example deployments/.env

# 2. Запустить auth-service (должен быть доступен на localhost:8081)

# 3. Запустить gateway
make run
```

Gateway будет доступен на `http://localhost:8080`

### Docker

```bash
# Собрать образ
docker build -t api-gateway:latest .

# Запустить контейнер
docker run -p 8080:8080 \
  -e AUTH_SERVICE_URL=http://auth-service \
  -e JWT_JWKS_URL=http://auth-service/.well-known/jwks.json \
  api-gateway:latest
```

### Kubernetes (Minikube)

```bash
# 1. Применить манифесты
kubectl apply -f deployments/k8s/

# 2. Включить Ingress addon
minikube addons enable ingress

# 3. Добавить в /etc/hosts
echo "$(minikube ip) cloud-storage.local" | sudo tee -a /etc/hosts

# 4. Проверить доступность
curl http://cloud-storage.local/health
```

## Конфигурация

Все настройки через environment variables:

```bash
# Server
SERVER_PORT=8080
SERVER_READ_TIMEOUT=10s
SERVER_WRITE_TIMEOUT=15s

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
```

Полный список в `deployments/.env.example`

## JWT Validation

Gateway валидирует JWT токены используя JWKS от auth-service:

1. **Startup**: загружает JWKS при старте
2. **Background refresh**: обновляет ключи каждый час
3. **Request validation**: проверяет подпись токена из `Authorization: Bearer <token>`
4. **Context injection**: добавляет `user_id` и `email` в context

**Алгоритм**: Ed25519 (EdDSA)

## Мониторинг

### Health Checks

```bash
# Liveness (всегда 200 если gateway работает)
curl http://localhost:8080/health

# Readiness (проверяет upstream сервисы)
curl http://localhost:8080/ready
```

### Логи

Structured logging через `log/slog`:

- **LOCAL**: pretty colored output
- **DEV**: JSON с DEBUG level
- **PROD**: JSON с INFO level

## Разработка

### Структура проекта

```
.
├── cmd/api-gateway/        # Entry point
├── internal/
│   ├── infrastructure/
│   │   ├── client/         # JWKS client
│   │   └── config/         # Configuration
│   └── presentation/
│       └── http/
│           ├── handler/    # Proxy, health checks
│           ├── middleware/ # JWT, CORS, rate limit
│           ├── router.go   # Route registration
│           └── server.go   # HTTP server
├── pkg/common/logger/      # Shared logger
└── deployments/k8s/        # Kubernetes manifests
```

### Сборка

```bash
# Локальная сборка
make build

# Запуск
make run

# Очистка
make clean
```

## Технологии

- **Go**: 1.26.2
- **Router**: chi v5
- **JWT**: golang-jwt/jwt v5 (Ed25519)
- **Rate Limiting**: golang.org/x/time/rate
- **Config**: cleanenv
- **Logging**: log/slog (stdlib)

## Безопасность

- ✅ JWT validation через JWKS
- ✅ Rate limiting per IP
- ✅ Non-root Docker user
- ✅ Request size limits (1MB)
- ✅ Hop-by-hop headers filtering
- ✅ Graceful shutdown

## Производительность

- **Rate limit**: 100 req/s per IP, burst 200
- **Upstream timeout**: 30s
- **JWKS cache**: in-memory, refresh every 1h
- **Connection pooling**: default http.Client

## Roadmap

- [ ] Metrics (Prometheus)
- [ ] Distributed tracing (OpenTelemetry)
- [ ] Circuit breaker для upstream
- [ ] Request/response logging middleware
- [ ] API versioning strategy
- [ ] WebSocket support

## Лицензия

MIT
