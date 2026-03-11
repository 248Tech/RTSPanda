# RTSPanda — single-binary build (frontend + backend). Run from repo root.
$ErrorActionPreference = "Stop"
$WebEmbedDir = "backend\internal\api\web"

Write-Host "Building frontend..."
Set-Location frontend
npm run build
Set-Location ..

Write-Host "Copying frontend/dist to backend for embed..."
if (Test-Path $WebEmbedDir) { Remove-Item -Recurse -Force $WebEmbedDir }
New-Item -ItemType Directory -Force -Path $WebEmbedDir | Out-Null
Copy-Item -Path "frontend\dist\*" -Destination $WebEmbedDir -Recurse -Force
# Preserve .gitkeep so the directory is tracked when empty
New-Item -ItemType File -Path "$WebEmbedDir\.gitkeep" -Force | Out-Null

Write-Host "Building Go binary..."
Set-Location backend
go build -o rtspanda.exe ./cmd/rtspanda
Set-Location ..

Write-Host "Done. Binary: backend\rtspanda.exe"
