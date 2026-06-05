$ErrorActionPreference = "Stop"

$InfluxHost = if ($env:INFLUX_HOST) { $env:INFLUX_HOST } else { "http://localhost:8086" }
$AdminUser = if ($env:INFLUX_ADMIN_USER) { $env:INFLUX_ADMIN_USER } else { "admin" }
$AdminPass = if ($env:INFLUX_ADMIN_PASS) { $env:INFLUX_ADMIN_PASS } else { "admin123456" }
$OrgName = "power-twin"
$BucketName = "power_telemetry"
$RetentionDays = 30

Write-Host "=== InfluxDB Initialization ==="
Write-Host "Host: $InfluxHost"

$MaxRetries = 30
$RetryCount = 0
$Ready = $false

while (-not $Ready) {
    $RetryCount++
    if ($RetryCount -ge $MaxRetries) {
        Write-Host "InfluxDB not ready after $MaxRetries retries, exiting."
        exit 1
    }
    try {
        $response = Invoke-WebRequest -Uri "$InfluxHost/health" -UseBasicParsing -ErrorAction Stop
        if ($response.StatusCode -eq 200) {
            $Ready = $true
        }
    } catch {
        Write-Host "Waiting for InfluxDB... ($RetryCount/$MaxRetries)"
        Start-Sleep -Seconds 2
    }
}
Write-Host "InfluxDB is ready."

Write-Host "--- Setting up initial config ---"
try {
    & influx setup `
        --host $InfluxHost `
        --username $AdminUser `
        --password $AdminPass `
        --org $OrgName `
        --bucket $BucketName `
        --retention "${RetentionDays}d" `
        --force `
        2>$null
    Write-Host "Setup completed."
} catch {
    Write-Host "Setup already completed, skipping."
}

Write-Host "--- Creating API token ---"
try {
    $tokenOutput = & influx auth create `
        --host $InfluxHost `
        --token $AdminPass `
        --org $OrgName `
        --read-buckets `
        --write-buckets `
        --description "power-twin read/write token" `
        2>$null
    if ($tokenOutput) {
        $apiToken = ($tokenOutput -split "`n" | Select-Object -First 1) -replace '\s.*', ''
        Write-Host "API Token created: $apiToken"
    }
} catch {
    Write-Host "Token may already exist, listing existing tokens:"
    & influx auth list --host $InfluxHost --token $AdminPass --org $OrgName
}

Write-Host "--- Writing initial test data ---"
$timestamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$lineProtocol = @"
device_telemetry,device_id=SUB_L1_01,device_type=substation,line_id=L1 voltage=1500i,current=1200i,power=1800000i,temperature=42.5,load_rate=55.0 ${timestamp}000000000
device_telemetry,device_id=SUB_L1_02,device_type=substation,line_id=L1 voltage=1480i,current=980i,power=1450400i,temperature=39.8,load_rate=48.3 ${timestamp}000000000
device_telemetry,device_id=RECT_L1_01_01,device_type=rectifier,line_id=L1 voltage=1510i,current=650i,power=981500i,temperature=51.2,load_rate=62.0 ${timestamp}000000000
device_telemetry,device_id=RECT_L1_01_02,device_type=rectifier,line_id=L1 voltage=1495i,current=580i,power=867100i,temperature=48.7,load_rate=58.5 ${timestamp}000000000
device_telemetry,device_id=DCS_L1_01_01,device_type=dc_switchgear,line_id=L1 voltage=1490i,current=320i,power=476800i,temperature=35.1,load_rate=42.0 ${timestamp}000000000
device_telemetry,device_id=DCS_L1_01_02,device_type=dc_switchgear,line_id=L1 voltage=1505i,current=280i,power=421400i,temperature=33.6,load_rate=38.2 ${timestamp}000000000
"@

$tempFile = [System.IO.Path]::GetTempFileName()
$lineProtocol | Set-Content -Path $tempFile -NoNewline

try {
    & influx write `
        --host $InfluxHost `
        --token $AdminPass `
        --org $OrgName `
        --bucket $BucketName `
        --file $tempFile `
        2>$null
    Write-Host "Test data written successfully."
} catch {
    Write-Host "Warning: Failed to write test data."
} finally {
    Remove-Item -Path $tempFile -Force -ErrorAction SilentlyContinue
}

Write-Host "--- Verifying test data ---"
try {
    $queryResult = & influx query `
        --host $InfluxHost `
        --token $AdminPass `
        --org $OrgName `
        "from(bucket: `"$BucketName`") |> range(start: -1h) |> filter(fn: (r) => r._measurement == `"device_telemetry`") |> limit(n: 3)" `
        2>$null
    if ($queryResult) {
        Write-Host "Verification successful. Sample data:"
        $queryResult | Select-Object -First 10 | ForEach-Object { Write-Host $_ }
    }
} catch {
    Write-Host "Warning: Could not verify test data."
}

Write-Host "=== InfluxDB Initialization Complete ==="
