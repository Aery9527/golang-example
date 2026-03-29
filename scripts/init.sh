#!/usr/bin/env bash
set -euo pipefail

# ──────────────────────────────────────────────
# Go project structure initializer (Bash)
# Generates community-standard layout.
# Safe to re-run: existing files are never overwritten.
# ──────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

# Read module name from go.mod
if [ ! -f go.mod ]; then
  echo "ERROR: go.mod not found in $ROOT_DIR" >&2
  exit 1
fi
MODULE=$(head -1 go.mod | awk '{print $2}')
echo "Module: $MODULE"

# ── Helper: write file only if it doesn't exist ──
write_file() {
  local path="$1"
  local content="$2"
  if [ -f "$path" ]; then
    echo "  SKIP  $path (already exists)"
  else
    mkdir -p "$(dirname "$path")"
    printf '%s\n' "$content" > "$path"
    echo "  CREATE $path"
  fi
}

# ── Directories (empty ones get a .gitkeep) ──
dirs=(
  "cmd/app"
  "internal/config"
  "internal/handler"
  "internal/service"
  "internal/repository"
  "api"
  "build"
  "deployments"
  "docs"
  "test/integration"
)

for d in "${dirs[@]}"; do
  mkdir -p "$d"
done
echo "Directories created."

# ── cmd/app/main.go ──
write_file "cmd/app/main.go" 'package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}'

# ── internal/config/config.go ──
write_file "internal/config/config.go" 'package config

// Config holds application configuration.
type Config struct {
	AppName string
	Port    int
}'

# ── internal/handler/handler.go ──
write_file "internal/handler/handler.go" 'package handler

import "net/http"

// HealthCheck responds with a simple health status.
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}'

# ── internal/service/service.go ──
write_file "internal/service/service.go" 'package service

// Service contains business logic.
type Service struct{}

// NewService creates a new Service.
func NewService() *Service {
	return &Service{}
}'

# ── internal/repository/repository.go ──
write_file "internal/repository/repository.go" 'package repository

// Repository handles data access.
type Repository struct{}

// NewRepository creates a new Repository.
func NewRepository() *Repository {
	return &Repository{}
}'

# ── .env.example ──
write_file ".env.example" '# Application
APP_NAME=myapp
APP_PORT=8080

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=
DB_NAME=myapp'

# ── build/Dockerfile ──
write_file "build/Dockerfile" "FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/app ./cmd/app

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/app /bin/app
ENTRYPOINT [\"/bin/app\"]"

# ── .gitignore ──
write_file ".gitignore" '# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
/bin/
/dist/

# Test
*.test
*.out
coverage.html

# Dependency
/vendor/

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Env
.env
*.env.local'

# ── Makefile ──
write_file "Makefile" ".PHONY: build run test lint clean

APP_NAME := app
BUILD_DIR := ./bin

build:
	go build -o \$(BUILD_DIR)/\$(APP_NAME) ./cmd/app

run:
	go run ./cmd/app

test:
	go test ./... -v

lint:
	golangci-lint run ./...

clean:
	rm -rf \$(BUILD_DIR)"

# ── .gitkeep for empty dirs ──
for d in api deployments docs "test/integration"; do
  write_file "$d/.gitkeep" ""
done

# ── 清理：清空 README.md 並移除初始化腳本 ──
echo ""

> "README.md"
echo "  CLEAR  README.md"

rm -f "scripts/init.sh" "scripts/init.ps1"
echo "  DELETE scripts/init.sh"
echo "  DELETE scripts/init.ps1"

echo ""
echo "================================================"
echo "  專案結構初始化完成！"
echo "  請編輯 README.md 開始這個專案的開發。"
echo "================================================"
echo ""
