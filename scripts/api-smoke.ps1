param(
  [string]$BaseUrl = "http://localhost:8080"
)

$ErrorActionPreference = "Stop"

function Invoke-Json {
  param(
    [string]$Method,
    [string]$Url,
    [object]$Body = $null
  )

  $params = @{
    Method = $Method
    Uri = $Url
  }

  if ($null -ne $Body) {
    $params.ContentType = "application/json"
    $params.Body = ($Body | ConvertTo-Json -Depth 5)
  }

  Invoke-RestMethod @params
}

Write-Host "Health check..."
$health = Invoke-Json -Method GET -Url "$BaseUrl/api/v1/health"
$health | ConvertTo-Json

$name = "SmokeTest-" + [Guid]::NewGuid().ToString("N").Substring(0, 8)
$cameraBody = @{
  name = $name
  rtsp_url = "rtsp://example.local/test"
  enabled = $true
}

Write-Host "Creating camera..."
$created = Invoke-Json -Method POST -Url "$BaseUrl/api/v1/cameras" -Body $cameraBody
$created | ConvertTo-Json -Depth 5

Write-Host "Listing cameras..."
$list = Invoke-Json -Method GET -Url "$BaseUrl/api/v1/cameras"
$list | ConvertTo-Json -Depth 5

Write-Host "Fetching stream status..."
$stream = Invoke-Json -Method GET -Url "$BaseUrl/api/v1/cameras/$($created.id)/stream"
$stream | ConvertTo-Json -Depth 5

Write-Host "Updating camera..."
$updated = Invoke-Json -Method PUT -Url "$BaseUrl/api/v1/cameras/$($created.id)" -Body @{
  name = "$name-updated"
  enabled = $false
}
$updated | ConvertTo-Json -Depth 5

Write-Host "Deleting camera..."
Invoke-RestMethod -Method DELETE -Uri "$BaseUrl/api/v1/cameras/$($created.id)"

Write-Host "Smoke test complete."
