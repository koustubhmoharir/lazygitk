#Requires -Version 7.0

# Builds portable, optimized lazygit binaries into dist/<os>_<arch>/ when run
# natively on Windows (PowerShell 7+). Use scripts/build_dist.sh elsewhere.
# Usage: scripts/build_dist.ps1 [-Targets os/arch,...]
#   e.g. scripts/build_dist.ps1 -Targets windows/amd64,windows/arm64
# With no arguments, builds windows/amd64. Builds use a temporary GOCACHE that
# is removed afterwards, so no build cache is left behind. Binaries are built
# with buildSource=forkDist, which disables upstream's self-updater (see
# pkg/updates/updates.go).

param(
    [string[]]$Targets = @('windows/amd64')
)

$ErrorActionPreference = 'Stop'

$scriptDir = $PSScriptRoot
$repoRoot = Split-Path -Parent $scriptDir
Set-Location -LiteralPath $repoRoot

foreach ($target in $Targets) {
    if ($target -notmatch '^[^/]+/[^/]+$') {
        throw "Invalid target '$target'; expected os/arch, e.g. windows/arm64"
    }
}

$envVars = @{
    GOTOOLCHAIN = 'local'
    CGO_ENABLED = '0'
    GOFLAGS     = '-mod=vendor'
}

# Save the caller's environment so it can be restored in finally.
$savedEnv = @{}
foreach ($name in @('GOCACHE', 'GOOS', 'GOARCH') + $envVars.Keys) {
    $savedEnv[$name] = [System.Environment]::GetEnvironmentVariable($name)
}

$gocache = Join-Path ([System.IO.Path]::GetTempPath()) ("lazygit-gocache-" + [System.Guid]::NewGuid().ToString('N'))

try {
    New-Item -ItemType Directory -Path $gocache | Out-Null
    $env:GOCACHE = $gocache
    foreach ($name in $envVars.Keys) {
        [System.Environment]::SetEnvironmentVariable($name, $envVars[$name])
    }

    $version = 'unversioned'
    $commit = ''
    if (Get-Command git -ErrorAction SilentlyContinue) {
        $described = git describe --tags --always --dirty 2>$null
        if ($LASTEXITCODE -eq 0 -and $described) { $version = $described }
        $head = git rev-parse HEAD 2>$null
        if ($LASTEXITCODE -eq 0 -and $head) { $commit = $head }
    }
    $buildDate = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')

    $ldflags = "-s -w -X main.version=$version -X main.commit=$commit -X main.date=$buildDate -X main.buildSource=forkDist"

    foreach ($target in $Targets) {
        $goos, $goarch = $target -split '/', 2
        $binary = if ($goos -eq 'windows') { 'lazygit.exe' } else { 'lazygit' }
        $out = Join-Path (Join-Path 'dist' "${goos}_${goarch}") $binary

        New-Item -ItemType Directory -Path (Split-Path -Parent $out) -Force | Out-Null
        Write-Host "Building $out"
        $env:GOOS = $goos
        $env:GOARCH = $goarch
        go build -trimpath -ldflags $ldflags -o $out .
        if ($LASTEXITCODE -ne 0) {
            throw "Build failed for $target"
        }
        Write-Host "Built $out"
    }
}
finally {
    foreach ($name in $savedEnv.Keys) {
        [System.Environment]::SetEnvironmentVariable($name, $savedEnv[$name])
    }
    if (Test-Path -LiteralPath $gocache) {
        Remove-Item -LiteralPath $gocache -Recurse -Force
    }
}
