.PHONY: build run test lint clean

APP_NAME := app
BUILD_DIR := ./bin

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/app

run:
	go run ./cmd/app

test:
	go test ./... -v

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)
