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

# ── 清理：寫入 README.md 骨架說明並移除初始化腳本 ──
echo ""

cat > "README.md" << 'READMEEOF'
# <專案名稱>

> ⚠️ **TODO**：此 README 由初始化腳本自動產生，請將整份內容替換為本專案的具體說明。

---

## 初始化骨架說明

以下檔案由初始化腳本產生，用於展示此 template 預設的 logging 與 error-handling 風格。
理解後請依專案需求修改或移除。

### 範例鏈路

| 檔案 | 角色 | 展示內容 |
| --- | --- | --- |
| [`cmd/app/main.go`](cmd/app/main.go) | 應用程式入口 | 組裝 handler / service / repository；以 `logs.Info` / `logs.ErrorWith` 記錄生命週期事件 |
| [`internal/handler/handler.go`](internal/handler/handler.go) | 邊界層 | `Handle() error` 介面；錯誤直接回傳上層，不在此包裝 |
| [`internal/service/service.go`](internal/service/service.go) | 服務層 | `errc.ServiceExampleRun.Wrap(err, "...")` 示範下游錯誤包裝慣例 |
| [`internal/repository/repository.go`](internal/repository/repository.go) | 資料層 | `errc.RepositoryExampleLoad.New("...")` 示範根錯誤建立 |
| [`internal/config/config.go`](internal/config/config.go) | 設定結構 | 最小 `Config` struct 骨架 |
| [`pkg/errc/code.go`](pkg/errc/code.go) | Error code 定義 | `ServiceExampleRun`、`RepositoryExampleLoad` 為 example 專用 code，實作時以業務 code 取代 |

### 為什麼有這些 example 檔案？

- **展示錯誤層次**：根錯誤在 repository 層建立，向上層層 Wrap，使 stack trace 與 error chain 完整呈現
- **展示 logging 時機**：僅在 `main()` 記錄生命週期事件，業務層保持 pure error return
- **展示命名慣例**：`ExampleHandler` / `ExampleService` / `ExampleRepository` 作為命名範本，實作時以具體業務名稱取代

READMEEOF
echo "  WRITE  README.md"

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
