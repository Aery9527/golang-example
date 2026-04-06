param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$ExtraArgs = @()
)

$ErrorActionPreference = 'Stop'
& (Join-Path $PSScriptRoot 'go-test.ps1') -Mode dev @ExtraArgs
$exitCode = $LASTEXITCODE

exit [int]$exitCode
