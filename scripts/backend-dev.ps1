param(
  [string]$DataDir = ".\backend\data",
  [string]$Port = "8080"
)

$ErrorActionPreference = "Stop"

$env:DATA_DIR = $DataDir
$env:PORT = $Port

Push-Location ".\backend"
try {
  go run .\cmd\rtspanda
} finally {
  Pop-Location
}
