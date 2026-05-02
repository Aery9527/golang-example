#Requires -Version 5.1
<#
.SYNOPSIS
    Go project structure initializer (PowerShell).
    Generates community-standard layout.
    Safe to re-run: existing files are never overwritten.
#>

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RootDir = $PSScriptRoot
Push-Location $RootDir
try {

# ── Helper: write file only if it doesn't exist ──
function Write-FileIfNotExists {
    param(
        [string]$Path,
        [string]$Content
    )
    if (Test-Path $Path) {
        Write-Host "  SKIP  $Path (already exists)"
    } else {
        $dir = Split-Path -Parent $Path
        if ($dir -and -not (Test-Path $dir)) {
            New-Item -ItemType Directory -Path $dir -Force | Out-Null
        }
        Set-Content -Path $Path -Value $Content -Encoding UTF8 -NoNewline
        Write-Host "  CREATE $Path"
    }
}

# ── Directories ──
$dirs = @(
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

foreach ($d in $dirs) {
    if (-not (Test-Path $d)) {
        New-Item -ItemType Directory -Path $d -Force | Out-Null
    }
}
Write-Host "Directories created."

# ── cmd/app/main.go ──
Write-FileIfNotExists "cmd/app/main.go" @"
package main

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
}
"@

# ── internal/config/config.go ──
Write-FileIfNotExists "internal/config/config.go" @"
package config

// Config holds application configuration.
type Config struct {
	AppName string
	Port    int
}
"@

# ── internal/handler/handler.go ──
Write-FileIfNotExists "internal/handler/handler.go" @"
package handler

import "golan-example/internal/service"

type ExampleHandler struct {
	service *service.ExampleService
}

func NewExampleHandler(service *service.ExampleService) *ExampleHandler {
	return &ExampleHandler{service: service}
}

func (h *ExampleHandler) Handle() error {
	return h.service.Run()
}
"@

# ── internal/service/service.go ──
Write-FileIfNotExists "internal/service/service.go" @"
package service

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
}
"@

# ── internal/repository/repository.go ──
Write-FileIfNotExists "internal/repository/repository.go" @"
package repository

import "golan-example/pkg/errc"

type ExampleRepository struct{}

func NewExampleRepository() *ExampleRepository {
	return &ExampleRepository{}
}

func (r *ExampleRepository) Load() error {
	return errc.RepositoryExampleLoad.New("example repository is not implemented")
}
"@

# ── .env.example ──
Write-FileIfNotExists ".env.example" @"
# Application
APP_NAME=myapp
APP_PORT=8080

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=
DB_NAME=myapp
"@

# ── build/Dockerfile ──
Write-FileIfNotExists "build/Dockerfile" @"
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/app ./cmd/app

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/app /bin/app
ENTRYPOINT ["/bin/app"]
"@

# ── .gitignore ──
Write-FileIfNotExists ".gitignore" @"
# Binaries
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
*.env.local
"@

# ── Makefile ──
Write-FileIfNotExists "Makefile" @"
.PHONY: build run test lint clean

APP_NAME := app
BUILD_DIR := ./bin

build:
	go build -o `$(BUILD_DIR)/`$(APP_NAME) ./cmd/app

run:
	go run ./cmd/app

test:
	go test ./... -v

lint:
	golangci-lint run ./...

clean:
	rm -rf `$(BUILD_DIR)
"@

# ── .gitkeep for empty dirs ──
foreach ($d in @("api", "deployments", "docs", "test/integration")) {
    Write-FileIfNotExists "$d/.gitkeep" ""
}

# ── 清理：寫入 README.md 骨架說明並移除初始化腳本 ──
Write-Host ""
$readmeContent = @'
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

'@
Set-Content -Path "README.md" -Value $readmeContent -Encoding UTF8 -NoNewline
Write-Host "  WRITE  README.md"

if (Test-Path "docs/superpowers") {
    Remove-Item -Path "docs/superpowers" -Recurse -Force
    Write-Host "  DELETE docs/superpowers"
}

$scriptFiles = @("init.sh", "init.ps1")
foreach ($f in $scriptFiles) {
    if (Test-Path $f) {
        Remove-Item -Path $f -Force
        Write-Host "  DELETE $f"
    }
}

if (Test-Path "scripts/tests") {
    Get-ChildItem -Path "scripts/tests" -Force |
        Where-Object { $_.Name -ne "test_release_notes.py" } |
        Remove-Item -Recurse -Force
    Write-Host "  CLEAN scripts/tests (kept test_release_notes.py)"
}

Write-Host ""
$previousErrorActionPreference = $ErrorActionPreference
$gitProbeExitCode = 1
try {
    $ErrorActionPreference = "Continue"
    git -C $RootDir rev-parse --is-inside-work-tree > $null 2> $null
    $gitProbeExitCode = $LASTEXITCODE
} finally {
    $ErrorActionPreference = $previousErrorActionPreference
}
if ($gitProbeExitCode -eq 0) {
    & "$RootDir/scripts/install-git-hooks.ps1"
} else {
    Write-Host "  SKIP  scripts/install-git-hooks.ps1 (not a git repository)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "================================================" -ForegroundColor Cyan
Write-Host "  專案結構初始化完成！" -ForegroundColor Green
Write-Host "  請編輯 README.md 開始這個專案的開發。" -ForegroundColor Yellow
Write-Host "================================================" -ForegroundColor Cyan
Write-Host ""

} finally {
    Pop-Location
}
