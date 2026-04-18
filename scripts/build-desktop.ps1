# Build the Downstage Write desktop app on Windows.
#
# Mirrors `make desktop-build` on Linux/macOS so Windows contributors
# don't need GNU make. Passes the same ldflags-injected version string
# and produces the binary at cmd\downstage-write\build\bin\.
#
# Usage (from any directory):
#   .\scripts\build-desktop.ps1
#   .\scripts\build-desktop.ps1 -Mode dev    # wails dev (hot reload)
#   .\scripts\build-desktop.ps1 -Mode debug  # -debug -devtools build

param(
    [ValidateSet("build", "dev", "debug")]
    [string]$Mode = "build"
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$version = (& git -C $repoRoot describe --tags --always --dirty 2>$null)
if (-not $version) { $version = "dev" }

$ldflags = "-X github.com/jscaltreto/downstage/internal/desktop.Version=$version"

Push-Location (Join-Path $repoRoot "cmd\downstage-write")
try {
    switch ($Mode) {
        "dev" {
            Write-Host "Starting desktop app in dev mode (version $version)..."
            & wails dev -ldflags $ldflags
        }
        "debug" {
            Write-Host "Building desktop app in debug mode (version $version)..."
            & wails build -debug -devtools -ldflags $ldflags
        }
        default {
            Write-Host "Building desktop app (version $version)..."
            & wails build -ldflags $ldflags
        }
    }
    if ($LASTEXITCODE -ne 0) {
        throw "wails exited with code $LASTEXITCODE"
    }
}
finally {
    Pop-Location
}
