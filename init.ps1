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
		return
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

# ── 清理：清空 README.md 並移除初始化腳本 ──
Write-Host ""
Set-Content -Path "README.md" -Value "" -Encoding UTF8 -NoNewline
Write-Host "  CLEAR  README.md"

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

Write-Host ""
git -C $RootDir rev-parse --is-inside-work-tree *> $null
if ($LASTEXITCODE -eq 0) {
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
