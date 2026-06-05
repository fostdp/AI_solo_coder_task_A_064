#!/usr/bin/env bash
set -euo pipefail

INFLUX_HOST="${INFLUX_HOST:-http://localhost:8086}"
ADMIN_USER="${INFLUX_ADMIN_USER:-admin}"
ADMIN_PASS="${INFLUX_ADMIN_PASS:-admin123456}"
ORG_NAME="power-twin"
BUCKET_NAME="power_telemetry"
RETENTION_DAYS=30

echo "=== InfluxDB Initialization ==="
echo "Host: ${INFLUX_HOST}"

MAX_RETRIES=30
RETRY_COUNT=0
until curl -sf "${INFLUX_HOST}/health" > /dev/null 2>&1; do
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ "${RETRY_COUNT}" -ge "${MAX_RETRIES}" ]; then
        echo "InfluxDB not ready after ${MAX_RETRIES} retries, exiting."
        exit 1
    fi
    echo "Waiting for InfluxDB... (${RETRY_COUNT}/${MAX_RETRIES})"
    sleep 2
done
echo "InfluxDB is ready."

echo "--- Setting up initial config ---"
influx setup \
    --host "${INFLUX_HOST}" \
    --username "${ADMIN_USER}" \
    --password "${ADMIN_PASS}" \
    --org "${ORG_NAME}" \
    --bucket "${BUCKET_NAME}" \
    --retention "${RETENTION_DAYS}d" \
    --force \
    2>/dev/null || echo "Setup already completed, skipping."

echo "--- Creating API token ---"
TOKEN_OUTPUT=$(influx auth create \
    --host "${INFLUX_HOST}" \
    --token "${ADMIN_PASS}" \
    --org "${ORG_NAME}" \
    --read-buckets \
    --write-buckets \
    --description "power-twin read/write token" \
    2>/dev/null || echo "TOKEN_CREATE_FAILED")

if echo "${TOKEN_OUTPUT}" | grep -q "TOKEN_CREATE_FAILED"; then
    echo "Token may already exist, listing existing tokens:"
    influx auth list \
        --host "${INFLUX_HOST}" \
        --token "${ADMIN_PASS}" \
        --org "${ORG_NAME}"
else
    API_TOKEN=$(echo "${TOKEN_OUTPUT}" | awk '/^[a-zA-Z0-9]/ {print $1}')
    echo "API Token created: ${API_TOKEN}"
fi

echo "--- Writing initial test data ---"
cat <<EOF | influx write \
    --host "${INFLUX_HOST}" \
    --token "${ADMIN_PASS}" \
    --org "${ORG_NAME}" \
    --bucket "${BUCKET_NAME}"
device_telemetry,device_id=SUB_L1_01,device_type=substation,line_id=L1 voltage=1500i,current=1200i,power=1800000i,temperature=42.5,load_rate=55.0 $(date +%s)000000000
device_telemetry,device_id=SUB_L1_02,device_type=substation,line_id=L1 voltage=1480i,current=980i,power=1450400i,temperature=39.8,load_rate=48.3 $(date +%s)000000000
device_telemetry,device_id=RECT_L1_01_01,device_type=rectifier,line_id=L1 voltage=1510i,current=650i,power=981500i,temperature=51.2,load_rate=62.0 $(date +%s)000000000
device_telemetry,device_id=RECT_L1_01_02,device_type=rectifier,line_id=L1 voltage=1495i,current=580i,power=867100i,temperature=48.7,load_rate=58.5 $(date +%s)000000000
device_telemetry,device_id=DCS_L1_01_01,device_type=dc_switchgear,line_id=L1 voltage=1490i,current=320i,power=476800i,temperature=35.1,load_rate=42.0 $(date +%s)000000000
device_telemetry,device_id=DCS_L1_01_02,device_type=dc_switchgear,line_id=L1 voltage=1505i,current=280i,power=421400i,temperature=33.6,load_rate=38.2 $(date +%s)000000000
EOF

echo "--- Verifying test data ---"
QUERY_RESULT=$(influx query \
    --host "${INFLUX_HOST}" \
    --token "${ADMIN_PASS}" \
    --org "${ORG_NAME}" \
    "from(bucket: \"${BUCKET_NAME}\") |> range(start: -1h) |> filter(fn: (r) => r._measurement == \"device_telemetry\") |> limit(n: 3)" \
    2>/dev/null)

if [ -n "${QUERY_RESULT}" ]; then
    echo "Verification successful. Sample data:"
    echo "${QUERY_RESULT}" | head -10
else
    echo "Warning: Could not verify test data."
fi

echo "=== InfluxDB Initialization Complete ==="
