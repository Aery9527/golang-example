$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8

$originalLocation = Get-Location
Set-Location (Join-Path $PSScriptRoot '..')
try {
    git rev-parse --is-inside-work-tree *> $null
    if ($LASTEXITCODE -ne 0) {
        throw "current directory is not a git repository"
    }

    git config --local core.hooksPath .githooks
    if ($LASTEXITCODE -ne 0) {
        throw "git config failed: exit $LASTEXITCODE"
    }

    Write-Host "[OK] core.hooksPath set to .githooks" -ForegroundColor Green
    exit 0
} finally {
    Set-Location $originalLocation
}
