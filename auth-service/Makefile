APP_NAME := auth-service
CMD_PATH := cmd/auth-service/main.go

.PHONY: run build clean

# Запуск сервиса локально
run:
	go run $(CMD_PATH)

# Сборка бинарника
build:
	go build -o bin/$(APP_NAME) $(CMD_PATH)

# Очистка
clean:
	rm -rf bin/