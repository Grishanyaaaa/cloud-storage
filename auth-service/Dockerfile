# Stage 1: Build
FROM golang:1.26.2-alpine AS builder

# Устанавливаем рабочую директорию
WORKDIR /build

# Устанавливаем необходимые системные зависимости
RUN apk add --no-cache git ca-certificates

# Копируем файлы зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем бинарный файл
RUN CGO_ENABLED=0 GOOS=linux go build -o auth-service ./cmd/auth-service/main.go

# Stage 2: Runtime
FROM alpine:latest

# Устанавливаем сертификаты и часовой пояс
RUN apk add --no-cache ca-certificates tzdata

# Создаем непривилегированного пользователя
RUN adduser -D -u 1001 appuser
USER appuser

WORKDIR /app

# Копируем бинарник
COPY --from=builder /build/auth-service .

# Пробрасываем порт (согласно вашему конфигу 8080)
EXPOSE 8080

# Запуск приложения
ENTRYPOINT ["./auth-service"]
