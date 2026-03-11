param(
  [string]$BaseUrl = "http://localhost:8080"
)

Write-Host "Create camera:"
Write-Host @"
Invoke-RestMethod -Method POST -Uri '$BaseUrl/api/v1/cameras' -ContentType 'application/json' -Body '{"name":"Front Door","rtsp_url":"rtsp://username:password@camera-host:554/stream1","enabled":true}'
"@

Write-Host ""
Write-Host "List cameras:"
Write-Host "Invoke-RestMethod -Method GET -Uri '$BaseUrl/api/v1/cameras'"

Write-Host ""
Write-Host "Get one camera:"
Write-Host "Invoke-RestMethod -Method GET -Uri '$BaseUrl/api/v1/cameras/<camera-id>'"

Write-Host ""
Write-Host "Get stream info:"
Write-Host "Invoke-RestMethod -Method GET -Uri '$BaseUrl/api/v1/cameras/<camera-id>/stream'"
