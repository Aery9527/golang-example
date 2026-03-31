param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$ExtraArgs = @()
)

$ErrorActionPreference = 'Stop'
$exitCode = & (Join-Path $PSScriptRoot 'go-test.ps1') -Mode ci @ExtraArgs

if ($exitCode -is [System.Array]) {
    $exitCode = $exitCode[-1]
}
if ($null -eq $exitCode) {
    $exitCode = $LASTEXITCODE
}

exit [int]$exitCode
