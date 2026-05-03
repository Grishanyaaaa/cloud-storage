# Stage 1: Build
FROM golang:1.26.2-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o storage-service ./cmd/storage-service/main.go

# Stage 2: Runtime
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

RUN adduser -D -u 1001 appuser
USER appuser

WORKDIR /app

COPY --from=builder /build/storage-service .

EXPOSE 8081

ENTRYPOINT ["./storage-service"]
