$ErrorActionPreference = "Stop"

function Test-DockerReady {
  try {
    docker info | Out-Null
    return $true
  } catch {
    return $false
  }
}

if (-not (Test-DockerReady)) {
  $desktopExe = "C:\Program Files\Docker\Docker\Docker Desktop.exe"
  if (-not (Test-Path $desktopExe)) {
    throw "Docker Desktop is not installed. Install Docker Desktop first."
  }

  Write-Host "Starting Docker Desktop..." -ForegroundColor Yellow
  Start-Process $desktopExe | Out-Null

  $maxWaitSeconds = 120
  $elapsed = 0
  while (-not (Test-DockerReady)) {
    Start-Sleep -Seconds 2
    $elapsed += 2
    if ($elapsed -ge $maxWaitSeconds) {
      throw "Docker engine did not become ready within $maxWaitSeconds seconds."
    }
  }
}

docker compose up --build -d
Write-Host "RTSPanda is running at http://localhost:8080" -ForegroundColor Green
