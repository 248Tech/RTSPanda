# RTSPanda — Windows one-line install (PowerShell 7)
# Installs Git, Go, Node.js (if missing), builds the app, optionally downloads mediamtx.
# Run from repo root, or pass -CloneFirst to clone the repo first.

param(
    [switch]$CloneFirst,
    [switch]$DownloadMediamtx,
    [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"

function Refresh-EnvPath {
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
}

function Test-Command($name) {
    $null -ne (Get-Command $name -ErrorAction SilentlyContinue)
}

Write-Host "RTSPanda — Windows install" -ForegroundColor Cyan

if ($CloneFirst) {
    $target = "RTSPanda"
    if (Test-Path $target) {
        Write-Host "Directory $target already exists. Using it." -ForegroundColor Yellow
        Set-Location $target
    } else {
        Write-Host "Cloning repository..."
        git clone https://github.com/248Tech/RTSPanda.git $target
        Set-Location $target
    }
}

$root = Get-Location
if (-not (Test-Path "build.ps1")) {
    Write-Host "Run this script from the RTSPanda repo root (where build.ps1 is)." -ForegroundColor Red
    exit 1
}

# Install tools via winget if missing
Refresh-EnvPath

if (-not (Test-Command "git")) {
    Write-Host "Installing Git..."
    winget install --id Git.Git -e --silent --accept-package-agreements
    Refresh-EnvPath
}
if (-not (Test-Command "go")) {
    Write-Host "Installing Go..."
    winget install --id GoLang.Go -e --silent --accept-package-agreements
    Refresh-EnvPath
}
if (-not (Test-Command "node")) {
    Write-Host "Installing Node.js LTS..."
    winget install --id OpenJS.NodeJS.LTS -e --silent --accept-package-agreements
    Refresh-EnvPath
}

# Ensure PATH is fresh for this session
Refresh-EnvPath

if (-not (Test-Command "go") -or -not (Test-Command "node")) {
    Write-Host "Go or Node not found after install. Close this window, open a new PowerShell 7, and run:" -ForegroundColor Yellow
    Write-Host "  cd '$root'; .\build.ps1" -ForegroundColor White
    exit 1
}

if ($DownloadMediamtx) {
    $mediamtxDir = Join-Path $root "mediamtx"
    $exePath = Join-Path $mediamtxDir "mediamtx.exe"
    if (Test-Path $exePath) {
        Write-Host "mediamtx.exe already present in mediamtx\" -ForegroundColor Green
    } else {
        Write-Host "Downloading mediamtx (Windows amd64)..."
        try {
            $releases = Invoke-RestMethod -Uri "https://api.github.com/repos/bluenviron/mediamtx/releases/latest" -Headers @{ "User-Agent" = "RTSPanda" }
            $asset = $releases.assets | Where-Object { $_.name -match "windows_amd64\.zip" } | Select-Object -First 1
            if (-not $asset) { throw "No windows_amd64 zip found" }
            $zipPath = Join-Path $env:TEMP "mediamtx.zip"
            Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $zipPath -UseBasicParsing
            New-Item -ItemType Directory -Force -Path $mediamtxDir | Out-Null
            Expand-Archive -Path $zipPath -DestinationPath $env:TEMP -Force
            $extracted = Get-ChildItem -Path $env:TEMP -Filter "mediamtx.exe" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
            if ($extracted) { Copy-Item $extracted.FullName -Destination $exePath -Force }
            Remove-Item $zipPath -ErrorAction SilentlyContinue
            Write-Host "mediamtx downloaded to mediamtx\mediamtx.exe" -ForegroundColor Green
        } catch {
            Write-Host "Could not download mediamtx: $_" -ForegroundColor Yellow
            Write-Host "Get it manually: https://github.com/bluenviron/mediamtx/releases" -ForegroundColor Gray
        }
    }
}

if (-not $SkipBuild) {
    Write-Host "Building RTSPanda..."
    & "$root\build.ps1"
}

Write-Host ""
Write-Host "Done. To run RTSPanda:" -ForegroundColor Green
Write-Host "  .\backend\rtspanda.exe" -ForegroundColor White
Write-Host "Then open http://localhost:8080 in your browser." -ForegroundColor Gray
