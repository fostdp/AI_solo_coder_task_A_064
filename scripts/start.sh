#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=== 城市轨道交通供电系统数字孪生平台 - Quick Start ==="
echo ""

echo "[1/4] Starting Docker Compose services..."
docker compose -f "${SCRIPT_DIR}/docker-compose.yml" up -d

echo ""
echo "[2/4] Waiting for InfluxDB to be ready..."
MAX_WAIT=60
WAITED=0
until curl -sf http://localhost:8086/health > /dev/null 2>&1; do
    WAITED=$((WAITED + 2))
    if [ "${WAITED}" -ge "${MAX_WAIT}" ]; then
        echo "InfluxDB did not become ready within ${MAX_WAIT}s"
        exit 1
    fi
    echo "  Waiting... (${WAITED}s/${MAX_WAIT}s)"
    sleep 2
done
echo "  InfluxDB is ready!"

echo ""
echo "[3/4] Running InfluxDB initialization..."
bash "${SCRIPT_DIR}/influxdb_init.sh"

echo ""
echo "[4/4] Checking service health..."
sleep 3

BACKEND_UP=false
FRONTEND_UP=false
MQTT_UP=false

if curl -sf http://localhost:8080/api/topology > /dev/null 2>&1 || curl -sf http://localhost:8080/ > /dev/null 2>&1; then
    BACKEND_UP=true
fi

if curl -sf http://localhost:3000 > /dev/null 2>&1; then
    FRONTEND_UP=true
fi

if nc -z localhost 1883 2>/dev/null; then
    MQTT_UP=true
fi

echo ""
echo "=== 服务状态 ==="
echo "  InfluxDB:    http://localhost:8086 (健康)"
echo "  Go后端:      http://localhost:8080 ($([ "$BACKEND_UP" = true ] && echo '健康' || echo '启动中...'))"
echo "  前端:        http://localhost:3000 ($([ "$FRONTEND_UP" = true ] && echo '健康' || echo '启动中...'))"
echo "  MQTT Broker: tcp://localhost:1883 ($([ "$MQTT_UP" = true ] && echo '健康' || echo '启动中...'))"
echo "  IEC61850端口: tcp://localhost:61850"
echo ""
echo "=== 访问地址 ==="
echo "  前端界面:    http://localhost:3000"
echo "  后端API:     http://localhost:8080/api/topology"
echo "  InfluxDB UI: http://localhost:8086"
echo "  InfluxDB登录: admin / admin123456"
echo ""
echo "=== 停止服务 ==="
echo "  docker compose -f ${SCRIPT_DIR}/docker-compose.yml down"
echo ""
echo "=== 启动完成 ==="
