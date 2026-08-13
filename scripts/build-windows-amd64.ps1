param(
    [switch]$Installer,
    [string[]]$WailsArgs = @()
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Split-Path -Parent $ScriptDir
if (-not (Get-Command wails -ErrorAction SilentlyContinue)) {
    throw "wails was not found on PATH. Install Wails CLI before building."
}

Push-Location $RepoRoot
try {
    Write-Host "HistoryLibHelper Windows AMD64 build"
    Write-Host "Repository: $RepoRoot"
    Write-Host ""

    $BuildArgs = @("build", "-clean", "-trimpath", "-platform", "windows/amd64")
    if ($Installer) {
        $BuildArgs += "-nsis"
    }
    $BuildArgs += $WailsArgs
    & wails @BuildArgs
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }

    & go build -trimpath -o build/bin/hlz-export.exe ./cmd/hlz-export
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
} finally {
    Pop-Location
}
