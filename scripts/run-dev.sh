#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PORT="${TELEMETRY_DEV_PORT:-18080}"
BIN="${TMPDIR:-/tmp}/telemetry-alert-dev"
LOG="${TMPDIR:-/tmp}/telemetry-alert-dev.log"
DB="${TMPDIR:-/tmp}/telemetry-alert-dev.db"

cd "$ROOT"
go build -o "$BIN" ./cmd/telemetry-alert

cleanup() {
  if [[ -n "${PID:-}" ]] && kill -0 "$PID" 2>/dev/null; then
    kill -TERM "$PID" 2>/dev/null || true
    wait "$PID" 2>/dev/null || true
  fi
  rm -f "$BIN" "$LOG" "$DB" "$DB-wal" "$DB-shm"
}
trap cleanup EXIT

DATABASE_SQLITE_PATH="$DB" SERVER_ADDR=":$PORT" ALERT_EVALUATION_INTERVAL_SECONDS=1 \
  "$BIN" -config configs/config.yaml >"$LOG" 2>&1 &
PID=$!

echo "waiting for service on port $PORT"
for _ in $(seq 1 100); do
  if curl -fsS "http://127.0.0.1:$PORT/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 0.1
done

echo "== /healthz =="
curl -fsS "http://127.0.0.1:$PORT/healthz"
echo
echo "== /readyz =="
curl -fsS "http://127.0.0.1:$PORT/readyz"
echo

tenant_response="$(curl -fsS -X POST "http://127.0.0.1:$PORT/v1/tenants" \
  -H 'Content-Type: application/json' \
  -d '{"name":"dev-tenant"}')"
tenant_id="$(printf '%s' "$tenant_response" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)"
echo "== create tenant =="
printf '%s\n' "$tenant_response"

device_response="$(curl -fsS -X POST "http://127.0.0.1:$PORT/v1/tenants/$tenant_id/devices" \
  -H 'Content-Type: application/json' \
  -d '{"name":"pump-01","type":"pump","serial_number":"PUMP-0001","location":"plant-a","tags":{"area":"east"}}')"
device_id="$(printf '%s' "$device_response" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)"
token="$(printf '%s' "$device_response" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)"
echo "== create device =="
printf '%s\n' "$device_response"

rule_response="$(curl -fsS -X POST "http://127.0.0.1:$PORT/v1/tenants/$tenant_id/rules" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"high-temp\",\"device_id\":\"$device_id\",\"metric_name\":\"temperature\",\"operator\":\"gt\",\"threshold\":80,\"window\":\"5m\",\"duration\":\"0s\",\"level\":\"warning\",\"cooldown\":\"1m\",\"channels\":[\"log\"]}")"
rule_id="$(printf '%s' "$rule_response" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)"
echo "== create alert rule =="
printf '%s\n' "$rule_response"

NOW="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
NOW_PLUS_5="$(date -u -v+5S +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '+5 seconds' +%Y-%m-%dT%H:%M:%SZ)"
RANGE_START="$(date -u -v-10M +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '10 minutes ago' +%Y-%m-%dT%H:%M:%SZ)"
RANGE_END="$(date -u -v+2H +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '+2 hours' +%Y-%m-%dT%H:%M:%SZ)"

echo "== ingest single point =="
curl -fsS -X POST "http://127.0.0.1:$PORT/v1/ingest" \
  -H "Authorization: Bearer $token" \
  -H 'Content-Type: application/json' \
  -d "{\"metric_name\":\"temperature\",\"value\":85.5,\"timestamp\":\"$NOW\",\"idempotency_key\":\"dev-check-1\"}"
echo

echo "== ingest batch =="
curl -fsS -X POST "http://127.0.0.1:$PORT/v1/ingest/batch" \
  -H "Authorization: Bearer $token" \
  -H 'Content-Type: application/json' \
  -d "{\"points\":[{\"metric_name\":\"temperature\",\"value\":86.0,\"timestamp\":\"$NOW_PLUS_5\"},{\"metric_name\":\"temperature\",\"value\":84.0,\"timestamp\":\"$NOW\"}]}"
echo

sleep 4
echo "== query raw telemetry =="
curl -fsS "http://127.0.0.1:$PORT/v1/tenants/$tenant_id/telemetry/raw?device_id=$device_id&metric_name=temperature&start=$RANGE_START&end=$RANGE_END&limit=100"
echo
echo "== query aggregates =="
curl -fsS "http://127.0.0.1:$PORT/v1/tenants/$tenant_id/telemetry/aggregates?device_id=$device_id&metric_name=temperature&granularity=5m&start=$RANGE_START&end=$RANGE_END"
echo
echo "== query alert events =="
curl -fsS "http://127.0.0.1:$PORT/v1/tenants/$tenant_id/events?rule_id=$rule_id"
echo

echo "== stopping service =="
kill -TERM "$PID"
wait "$PID"
PID=""
echo "service stopped"
