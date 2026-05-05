APP_NAME := ai-service
CMD_PATH := cmd/ai-service/main.go

.PHONY: run build clean test vet lint

run:
	go run $(CMD_PATH)

build:
	go build -o bin/$(APP_NAME) $(CMD_PATH)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf bin/
