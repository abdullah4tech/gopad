[CmdletBinding()]
param(
    [string]$Output = "gopad.exe"
)

$ErrorActionPreference = "Stop"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go is required. Install it from https://go.dev/dl/ and try again."
}

if (-not (Get-Command gcc -ErrorAction SilentlyContinue)) {
    throw @"
GCC is required because GoPad uses the CGO-based go-sqlite3 driver.
Install a MinGW-w64 distribution, open a new PowerShell window, and try again.
For example: winget install --id BrechtSanders.WinLibs.POSIX.UCRT -e
"@
}

$version = git describe --tags --always --dirty 2>$null
if (-not $version) {
    $version = "dev"
}

$env:CGO_ENABLED = "1"
go build `
    -tags "desktop,production" `
    -ldflags "-s -w -H windowsgui -X main.version=$version" `
    -o $Output `
    .

if ($LASTEXITCODE -ne 0) {
    throw "GoPad build failed with exit code $LASTEXITCODE."
}

Write-Host "Built $Output ($version)"
