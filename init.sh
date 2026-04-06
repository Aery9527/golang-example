#!/usr/bin/env bash
set -euo pipefail

# ──────────────────────────────────────────────
# Go project structure initializer (Bash)
# Generates community-standard layout.
# Safe to re-run: existing files are never overwritten.
# ──────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$SCRIPT_DIR"
cd "$ROOT_DIR"

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

import (
	"golan-example/internal/handler"
	"golan-example/internal/logs"
	"golan-example/internal/repository"
	"golan-example/internal/service"
)

func main() {
	h := handler.NewExampleHandler(
		service.NewExampleService(
			repository.NewExampleRepository(),
		),
	)

	logs.Info("application starting", func() []any {
		return []any{"component", "app"}
	})

	if err := h.Handle(); err != nil {
		logs.ErrorWith("application stopped", func() (error, []any) {
			return err, []any{"component", "app"}
		})
	}

	logs.Info("application finished", func() []any {
		return []any{"component", "app"}
	})
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

import "golan-example/internal/service"

type ExampleHandler struct {
	service *service.ExampleService
}

func NewExampleHandler(service *service.ExampleService) *ExampleHandler {
	return &ExampleHandler{service: service}
}

func (h *ExampleHandler) Handle() error {
	return h.service.Run()
}'

# ── internal/service/service.go ──
write_file "internal/service/service.go" 'package service

import (
	"golan-example/internal/repository"
	"golan-example/pkg/errc"
)

type ExampleService struct {
	repository *repository.ExampleRepository
}

func NewExampleService(repository *repository.ExampleRepository) *ExampleService {
	return &ExampleService{repository: repository}
}

func (s *ExampleService) Run() error {
	if err := s.repository.Load(); err != nil {
		return errc.ServiceExampleRun.Wrap(err, "run example service")
	}
	return nil
}'

# ── internal/repository/repository.go ──
write_file "internal/repository/repository.go" 'package repository

import "golan-example/pkg/errc"

type ExampleRepository struct{}

func NewExampleRepository() *ExampleRepository {
	return &ExampleRepository{}
}

func (r *ExampleRepository) Load() error {
	return errc.RepositoryExampleLoad.New("example repository is not implemented")
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

rm -rf "docs/superpowers"
echo "  DELETE docs/superpowers"

rm -f "init.sh" "init.ps1"
echo "  DELETE init.sh"
echo "  DELETE init.ps1"

if [ -d "scripts/tests" ]; then
  find "scripts/tests" -mindepth 1 -maxdepth 1 ! -name "test_release_notes.py" -exec rm -rf {} +
  echo "  CLEAN scripts/tests (kept test_release_notes.py)"
fi

echo ""
if git -C "$ROOT_DIR" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  bash "$ROOT_DIR/scripts/install-git-hooks.sh"
else
  echo "  SKIP  scripts/install-git-hooks.sh (not a git repository)"
fi

echo ""
echo "================================================"
echo "  專案結構初始化完成！"
echo "  請編輯 README.md 開始這個專案的開發。"
echo "================================================"
echo ""
