$InfluxUrl = "http://localhost:8086"
$Org = "power-twin"
$Token = ""

Write-Host "Waiting for InfluxDB..."
while ($true) {
    try {
        $r = Invoke-WebRequest -Uri "$InfluxUrl/health" -UseBasicParsing -ErrorAction Stop
        if ($r.Content -match "pass") { break }
    } catch {}
    Start-Sleep -Seconds 2
}
Write-Host "InfluxDB is ready."

try {
    $body = @{
        username = "admin"
        password = "power-twin-admin-2024"
        org = $Org
        bucket = "telemetry"
        retentionDurationSeconds = 2592000
    } | ConvertTo-Json

    $resp = Invoke-RestMethod -Uri "$InfluxUrl/api/v2/setup" -Method Post -Body $body -ContentType "application/json"
    $Token = $resp.token
} catch {
    Write-Host "Already configured, using env token"
    $Token = if ($env:INFLUXDB_ADMIN_TOKEN) { $env:INFLUXDB_ADMIN_TOKEN } else { "my-token" }
}

Write-Host "Token: $($Token.Substring(0,10))..."

$headers = @{ "Authorization" = "Token $Token"; "Content-Type" = "application/json" }

try {
    $body = '{"name":"telemetry_downsampled","orgID":"power-twin","retentionRules":[{"type":"expire","everySeconds":7776000}]}'
    Invoke-RestMethod -Uri "$InfluxUrl/api/v2/buckets" -Method Post -Body $body -Headers $headers
} catch { Write-Host "Bucket telemetry_downsampled may already exist" }

try {
    $body = '{"name":"telemetry_archive","orgID":"power-twin","retentionRules":[{"type":"expire","everySeconds":31536000}]}'
    Invoke-RestMethod -Uri "$InfluxUrl/api/v2/buckets" -Method Post -Body $body -Headers $headers
} catch { Write-Host "Bucket telemetry_archive may already exist" }

Write-Host "InfluxDB initialization complete."
