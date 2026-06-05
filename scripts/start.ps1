$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

Write-Host "=== 城市轨道交通供电系统数字孪生平台 - Quick Start ==="
Write-Host ""

Write-Host "[1/4] Starting Docker Compose services..."
$composeCmd = $null
try {
    docker compose version 2>$null | Out-Null
    $composeCmd = "docker compose"
} catch {
    try {
        docker-compose version 2>$null | Out-Null
        $composeCmd = "docker-compose"
    } catch {
        Write-Host "Error: docker compose or docker-compose not found."
        exit 1
    }
}

& cmd /c "$composeCmd -f `"$ScriptDir\docker-compose.yml`" up -d"
Write-Host ""

Write-Host "[2/4] Waiting for InfluxDB to be ready..."
$MaxWait = 60
$Waited = 0
$Ready = $false
while (-not $Ready) {
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:8086/health" -UseBasicParsing -ErrorAction Stop
        if ($response.StatusCode -eq 200) {
            $Ready = $true
        }
    } catch {
        $Waited += 2
        if ($Waited -ge $MaxWait) {
            Write-Host "InfluxDB did not become ready within ${MaxWait}s"
            exit 1
        }
        Write-Host "  Waiting... ($Waited`s/$MaxWait`s)"
        Start-Sleep -Seconds 2
    }
}
Write-Host "  InfluxDB is ready!"
Write-Host ""

Write-Host "[3/4] Running InfluxDB initialization..."
& powershell -ExecutionPolicy Bypass -File "$ScriptDir\influxdb_init.ps1"
Write-Host ""

Write-Host "[4/4] Checking service health..."
Start-Sleep -Seconds 3

$BackendUp = $false
$FrontendUp = $false

try {
    $r = Invoke-WebRequest -Uri "http://localhost:8080/" -UseBasicParsing -TimeoutSec 3 -ErrorAction Stop
    $BackendUp = $true
} catch {}

try {
    $r = Invoke-WebRequest -Uri "http://localhost:3000" -UseBasicParsing -TimeoutSec 3 -ErrorAction Stop
    $FrontendUp = $true
} catch {}

$backendStatus = if ($BackendUp) { "健康" } else { "启动中..." }
$frontendStatus = if ($FrontendUp) { "健康" } else { "启动中..." }

Write-Host ""
Write-Host "=== 服务状态 ==="
Write-Host "  InfluxDB:    http://localhost:8086 (健康)"
Write-Host "  Go后端:      http://localhost:8080 ($backendStatus)"
Write-Host "  前端:        http://localhost:3000 ($frontendStatus)"
Write-Host "  MQTT Broker: tcp://localhost:1883"
Write-Host "  IEC61850端口: tcp://localhost:61850"
Write-Host ""
Write-Host "=== 访问地址 ==="
Write-Host "  前端界面:    http://localhost:3000"
Write-Host "  后端API:     http://localhost:8080/api/topology"
Write-Host "  InfluxDB UI: http://localhost:8086"
Write-Host "  InfluxDB登录: admin / admin123456"
Write-Host ""
Write-Host "=== 停止服务 ==="
Write-Host "  $composeCmd -f `"$ScriptDir\docker-compose.yml`" down"
Write-Host ""
Write-Host "=== 启动完成 ==="
