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

function ConvertTo-PowerShellQuotedString {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Value
    )

    return "'" + ($Value -replace "'", "''") + "'"
}

function Test-BoolTruthy {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Value
    )

    switch ($Value.ToLowerInvariant()) {
        '1' { return $true }
        't' { return $true }
        'true' { return $true }
        '0' { return $false }
        'f' { return $false }
        'false' { return $false }
        default { return $null }
    }
}

function Get-SelectedStdoutFile {
    param(
        [string[]]$ArgList = @(),

        [Parameter(Mandatory = $true)]
        [string]$ArtifactDir
    )

    $stdoutFile = Join-Path $ArtifactDir 'stdout.log'
    foreach ($arg in $ArgList) {
        if ($arg -eq '-json') {
            $stdoutFile = Join-Path $ArtifactDir 'stdout.jsonl'
            continue
        }

        if ($arg -match '^-json=(.+)$') {
            $truthy = Test-BoolTruthy $Matches[1]
            if ($null -ne $truthy) {
                $stdoutFile = if ($truthy) {
                    Join-Path $ArtifactDir 'stdout.jsonl'
                } else {
                    Join-Path $ArtifactDir 'stdout.log'
                }
            }
        }
    }

    return $stdoutFile
}

function Test-CoverpkgValue {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Value
    )

    foreach ($part in ($Value -split ',')) {
        $scope = $part.Trim()
        if ($scope -ne './internal/...' -and $scope -ne './pkg/...') {
            return "invalid extra arg '$Value': -coverpkg must stay within ./internal/... and ./pkg/... scopes"
        }
    }

    return $null
}

function Test-ExtraArgs {
    param(
        [string[]]$ArgList = @()
    )

    $expectingValueFor = $null
    foreach ($arg in $ArgList) {
        if ($null -ne $expectingValueFor) {
            if ($expectingValueFor -eq '-coverpkg') {
                $message = Test-CoverpkgValue $arg
                if ($null -ne $message) {
                    return $message
                }
            }

            $expectingValueFor = $null
            continue
        }

        if (-not $arg.StartsWith('-')) {
            return "invalid extra arg '$arg': package patterns are fixed to ./internal/... and ./pkg/...; extra args may only be flags or flag values"
        }

        switch -regex ($arg) {
            '^-args(=|$)' {
                return "invalid extra arg '$arg': -args is not supported because the runner appends fixed package targets after user args"
            }
            '^-coverprofile(=|$)' {
                return "invalid extra arg '$arg': -coverprofile is reserved for the runner's dev-mode coverage artifact path"
            }
            '^-coverpkg$' {
                $expectingValueFor = '-coverpkg'
                continue
            }
            '^-coverpkg=(.+)$' {
                $message = Test-CoverpkgValue $Matches[1]
                if ($null -ne $message) {
                    return $message
                }
                continue
            }
            '^-asmflags$|^-bench$|^-benchtime$|^-blockprofile$|^-blockprofilerate$|^-count$|^-covermode$|^-cpu$|^-cpuprofile$|^-exec$|^-fuzz$|^-fuzzminimizetime$|^-fuzztime$|^-gcflags$|^-ldflags$|^-list$|^-memprofile$|^-memprofilerate$|^-mod$|^-modfile$|^-mutexprofile$|^-mutexprofilefraction$|^-o$|^-outputdir$|^-overlay$|^-p$|^-parallel$|^-pkgdir$|^-pgo$|^-run$|^-shuffle$|^-skip$|^-tags$|^-timeout$|^-toolexec$|^-trace$|^-vet$|^-vettool$' {
                $expectingValueFor = $arg
                continue
            }
            default {
                continue
            }
        }
    }

    if ($null -ne $expectingValueFor) {
        if ($expectingValueFor -eq '-coverpkg') {
            return "invalid extra arg '$expectingValueFor': -coverpkg requires a value"
        }

        return "invalid extra arg '$expectingValueFor': expected a value; package patterns are fixed to ./internal/... and ./pkg/..."
    }

    return $null
}

function New-CommandText {
    param(
        [Parameter(Mandatory = $true)]
        [string]$RepoRoot,

        [Parameter(Mandatory = $true)]
        [string]$GoExecutable,

        [Parameter(Mandatory = $true)]
        [string[]]$GoArgs,

        [Parameter(Mandatory = $false)]
        [string]$BlockedReason
    )

    $lines = @()
    if ($BlockedReason) {
        $lines += "# blocked before go test ran: $BlockedReason"
    }

    $quotedRepoRoot = ConvertTo-PowerShellQuotedString $RepoRoot
    $quotedGo = ConvertTo-PowerShellQuotedString $GoExecutable
    $command = "Set-Location -LiteralPath $quotedRepoRoot; & $quotedGo"
    foreach ($arg in $GoArgs) {
        $command += ' ' + (ConvertTo-PowerShellQuotedString $arg)
    }

    $lines += $command
    return ($lines -join [Environment]::NewLine)
}

$originalLocation = Get-Location
$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$artifactDir = Join-Path $repoRoot "test-output\$Mode-test"
$stdoutFile = $null
$stderrFile = $null
$commandFile = $null
$exitCodeFile = $null
$coverageFile = $null
$coverageSummaryFile = $null
$scriptExitCode = 0

try {
    Set-Location $repoRoot

    if (Test-Path $artifactDir) {
        Remove-Item -Recurse -Force $artifactDir
    }
    New-Item -ItemType Directory -Force -Path $artifactDir | Out-Null

    $stdoutFile = Get-SelectedStdoutFile -ArgList $ExtraArgs -ArtifactDir $artifactDir
    $stderrFile = Join-Path $artifactDir 'stderr.log'
    $commandFile = Join-Path $artifactDir 'command.txt'
    $exitCodeFile = Join-Path $artifactDir 'exit-code.txt'
    $coverageFile = Join-Path $artifactDir 'coverage.out'
    $coverageSummaryFile = Join-Path $artifactDir 'coverage-summary.txt'

    New-Item -ItemType File -Force -Path $stdoutFile | Out-Null
    New-Item -ItemType File -Force -Path $stderrFile | Out-Null

    $goCommand = Get-Command go -CommandType Application -ErrorAction Stop | Select-Object -First 1
    $goExecutable = $goCommand.Source
    if (-not $goExecutable) {
        $goExecutable = $goCommand.Path
    }

    if (-not $goExecutable) {
        throw "go executable not found in PATH"
    }

    $testRoots = @('internal', 'pkg')
    $targets = foreach ($root in $testRoots) { "./$root/..." }

    $goArgs = @('test')
    if ($Mode -eq 'ci') {
        $goArgs += '-short'
    }
    if ($Mode -eq 'dev') {
        $goArgs += "-coverprofile=$coverageFile"
    }
    $goArgs += $ExtraArgs
    $goArgs += $targets

    $validationMessage = Test-ExtraArgs -ArgList $ExtraArgs
    $commandText = New-CommandText -RepoRoot $repoRoot -GoExecutable $goExecutable -GoArgs $goArgs -BlockedReason $validationMessage

    if ($null -ne $validationMessage) {
        $scriptExitCode = 2
        Set-Content -Path $commandFile -Value $commandText -NoNewline
        Set-Content -Path $stderrFile -Value $validationMessage
        if (Test-Path $stdoutFile) {
            Clear-Content -Path $stdoutFile
        }
    } else {
        & $goExecutable @goArgs 1> $stdoutFile 2> $stderrFile
        $scriptExitCode = $LASTEXITCODE
        Set-Content -Path $commandFile -Value $commandText -NoNewline
    }

    if ((Test-Path $stdoutFile) -and ((Get-Item $stdoutFile).Length -gt 0)) {
        Get-Content -Path $stdoutFile | ForEach-Object { Write-Output $_ }
    }
    if ((Test-Path $stderrFile) -and ((Get-Item $stderrFile).Length -gt 0)) {
        Get-Content -Path $stderrFile | ForEach-Object { [Console]::Error.WriteLine($_) }
    }

    if ($Mode -eq 'dev' -and (Test-Path $coverageFile)) {
        & $goExecutable tool cover "-func=$coverageFile" | Tee-Object -FilePath $coverageSummaryFile | ForEach-Object { Write-Output $_ }
        if ($LASTEXITCODE -ne 0 -and $scriptExitCode -eq 0) {
            $scriptExitCode = $LASTEXITCODE
        }
    }
} catch {
    $scriptExitCode = if ($_.Exception -is [System.Management.Automation.CommandNotFoundException]) { 127 } else { 1 }

    if (-not $stdoutFile) {
        $stdoutFile = Join-Path $artifactDir 'stdout.log'
    }
    if (-not $stderrFile) {
        $stderrFile = Join-Path $artifactDir 'stderr.log'
    }
    if (-not $commandFile) {
        $commandFile = Join-Path $artifactDir 'command.txt'
    }
    if (-not $exitCodeFile) {
        $exitCodeFile = Join-Path $artifactDir 'exit-code.txt'
    }

    New-Item -ItemType Directory -Force -Path $artifactDir | Out-Null
    New-Item -ItemType File -Force -Path $stdoutFile | Out-Null
    New-Item -ItemType File -Force -Path $stderrFile | Out-Null

    $fallbackArgs = if ($null -ne $goArgs) { $goArgs } else { @('test') }
    $fallbackCommand = New-CommandText -RepoRoot $repoRoot -GoExecutable '<go executable unavailable>' -GoArgs $fallbackArgs -BlockedReason $_.Exception.Message
    Set-Content -Path $commandFile -Value $fallbackCommand -NoNewline
    Set-Content -Path $stderrFile -Value $_.Exception.Message
} finally {
    if ($null -ne $exitCodeFile) {
        Set-Content -Path $exitCodeFile -Value $scriptExitCode -NoNewline
    }

    Set-Location $originalLocation
}

exit [int]$scriptExitCode
