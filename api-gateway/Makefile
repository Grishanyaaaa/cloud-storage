APP_NAME := api-gateway
CMD_PATH := cmd/api-gateway/main.go

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
