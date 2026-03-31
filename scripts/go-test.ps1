param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('ci', 'dev')]
    [string]$Mode,

    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$ExtraArgs = @()
)

$ErrorActionPreference = 'Stop'
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8

$originalLocation = Get-Location
Set-Location (Join-Path $PSScriptRoot '..')
try {
    $testRoots = @('internal', 'pkg')
    $artifactDir = Join-Path 'test-output' "$Mode-test"
    New-Item -ItemType Directory -Force -Path $artifactDir | Out-Null

    $targets = @()
    foreach ($root in $testRoots) {
        $targets += "./$root/..."
    }

    $stdoutFile = Join-Path $artifactDir 'stdout.log'
    if ($ExtraArgs -contains '-json') {
        $stdoutFile = Join-Path $artifactDir 'stdout.jsonl'
    }

    $stderrFile = Join-Path $artifactDir 'stderr.log'
    $commandFile = Join-Path $artifactDir 'command.txt'
    $exitCodeFile = Join-Path $artifactDir 'exit-code.txt'
    $coverageFile = Join-Path $artifactDir 'coverage.out'
    $coverageSummaryFile = Join-Path $artifactDir 'coverage-summary.txt'

    $goArgs = @('test')
    if ($Mode -eq 'ci') {
        $goArgs += '-short'
    }
    if ($Mode -eq 'dev') {
        $goArgs += "-coverprofile=$coverageFile"
    }
    $goArgs += $ExtraArgs
    $goArgs += $targets

    "go $($goArgs -join ' ')" | Set-Content -Path $commandFile

    $process = Start-Process `
        -FilePath 'go' `
        -ArgumentList $goArgs `
        -NoNewWindow `
        -Wait `
        -PassThru `
        -RedirectStandardOutput $stdoutFile `
        -RedirectStandardError $stderrFile

    $exitCode = $process.ExitCode
    "$exitCode" | Set-Content -Path $exitCodeFile

    if (Test-Path $stdoutFile) {
        Get-Content $stdoutFile | ForEach-Object { Write-Host $_ }
    }
    if (Test-Path $stderrFile) {
        Get-Content $stderrFile | ForEach-Object { Write-Host $_ -ForegroundColor Red }
    }

    if ($Mode -eq 'dev' -and (Test-Path $coverageFile)) {
        & go tool cover "-func=$coverageFile" |
            Tee-Object -FilePath $coverageSummaryFile |
            ForEach-Object { Write-Host $_ }
        if ($LASTEXITCODE -ne 0) {
            throw "go tool cover failed: exit $LASTEXITCODE"
        }
    }

    $global:LASTEXITCODE = $exitCode
    return $exitCode
} finally {
    Set-Location $originalLocation
}
