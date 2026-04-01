param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$ExtraArgs = @()
)

$ErrorActionPreference = 'Stop'
& (Join-Path $PSScriptRoot 'go-test.ps1') -Mode ci @ExtraArgs
exit [int]$LASTEXITCODE
