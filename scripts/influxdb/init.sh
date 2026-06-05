#!/bin/bash
set -e

INFLUX_URL="http://localhost:8086"
ORG="power-twin"
TOKEN=""

echo "Waiting for InfluxDB to start..."
until curl -s "${INFLUX_URL}/health" | grep -q "pass"; do
  sleep 2
done
echo "InfluxDB is ready."

SETUP_RESPONSE=$(curl -s -X POST "${INFLUX_URL}/api/v2/setup" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "power-twin-admin-2024",
    "org": "'${ORG}'",
    "bucket": "telemetry",
    "retentionDurationSeconds": 2592000
  }') || echo "Already configured"

TOKEN=$(echo "$SETUP_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "InfluxDB already configured, fetching token..."
  TOKEN="${INFLUXDB_ADMIN_TOKEN:-my-token}"
fi

echo "Token obtained: ${TOKEN:0:10}..."

curl -s -X POST "${INFLUX_URL}/api/v2/buckets" \
  -H "Authorization: Token ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "telemetry_downsampled",
    "orgID": "'${ORG}'",
    "retentionRules": [{"type": "expire", "everySeconds": 7776000}]
  }' || echo "Bucket telemetry_downsampled may already exist"

curl -s -X POST "${INFLUX_URL}/api/v2/buckets" \
  -H "Authorization: Token ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "telemetry_archive",
    "orgID": "'${ORG}'",
    "retentionRules": [{"type": "expire", "everySeconds": 31536000}]
  }' || echo "Bucket telemetry_archive may already exist"

ORG_ID=$(curl -s "${INFLUX_URL}/api/v2/orgs?org=${ORG}" \
  -H "Authorization: Token ${TOKEN}" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

echo "Org ID: ${ORG_ID}"

curl -s -X POST "${INFLUX_URL}/api/v2/tasks" \
  -H "Authorization: Token ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "downsample-5m",
    "orgID": "'${ORG_ID}'",
    "flux": "option task = {name: \"downsample-5m\", every: 5m}\n\nfrom(bucket: \"telemetry\")\n  |> range(start: -5m)\n  |> filter(fn: (r) => r._measurement == \"device_telemetry\")\n  |> aggregateWindow(every: 5m, fn: mean, createEmpty: false)\n  |> to(bucket: \"telemetry_downsampled\", org: \"'${ORG}'\")",
    "status": "active",
    "every": "5m"
  }' || echo "Task downsample-5m may already exist"

curl -s -X POST "${INFLUX_URL}/api/v2/tasks" \
  -H "Authorization: Token ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "downsample-1h",
    "orgID": "'${ORG_ID}'",
    "flux": "option task = {name: \"downsample-1h\", every: 1h}\n\nfrom(bucket: \"telemetry_downsampled\")\n  |> range(start: -1h)\n  |> filter(fn: (r) => r._measurement == \"device_telemetry\")\n  |> aggregateWindow(every: 1h, fn: mean, createEmpty: false)\n  |> to(bucket: \"telemetry_archive\", org: \"'${ORG}'\")",
    "status": "active",
    "every": "1h"
  }' || echo "Task downsample-1h may already exist"

echo "InfluxDB initialization complete."
echo "Token: ${TOKEN}"
