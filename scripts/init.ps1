#Requires -Version 5.1
<#
.SYNOPSIS
    Go project structure initializer (PowerShell).
    Generates community-standard layout.
    Safe to re-run: existing files are never overwritten.
#>

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RootDir = Split-Path -Parent $PSScriptRoot
Push-Location $RootDir
try {

# Read module name from go.mod
if (-not (Test-Path "go.mod")) {
    Write-Error "go.mod not found in $RootDir"
    exit 1
}
$Module = (Get-Content "go.mod" -First 1) -replace '^module\s+', ''
Write-Host "Module: $Module"

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
    "pkg/logger"
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

import "fmt"

func main() {
	fmt.Println("Hello, World!")
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

import "net/http"

// HealthCheck responds with a simple health status.
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
"@

# ── internal/service/service.go ──
Write-FileIfNotExists "internal/service/service.go" @"
package service

// Service contains business logic.
type Service struct{}

// NewService creates a new Service.
func NewService() *Service {
	return &Service{}
}
"@

# ── internal/repository/repository.go ──
Write-FileIfNotExists "internal/repository/repository.go" @"
package repository

// Repository handles data access.
type Repository struct{}

// NewRepository creates a new Repository.
func NewRepository() *Repository {
	return &Repository{}
}
"@

# ── pkg/logger/logger.go ──
Write-FileIfNotExists "pkg/logger/logger.go" @"
package logger

import "log"

// Info logs an informational message.
func Info(msg string) {
	log.Println("[INFO]", msg)
}

// Error logs an error message.
func Error(msg string) {
	log.Println("[ERROR]", msg)
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

$scriptFiles = @("scripts/init.sh", "scripts/init.ps1")
foreach ($f in $scriptFiles) {
    if (Test-Path $f) {
        Remove-Item -Path $f -Force
        Write-Host "  DELETE $f"
    }
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
