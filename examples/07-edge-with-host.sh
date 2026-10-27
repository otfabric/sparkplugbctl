#!/usr/bin/env bash
# Full scenario: start a host, an edge node, and a listener in parallel.
# Demonstrates a complete Sparkplug B infrastructure on a single broker.
set -euo pipefail

BROKER="${BROKER:-tcp://localhost:1883}"

cleanup() {
  echo
  echo "Stopping all processes..."
  kill "${HOST_PID:-}" "${EDGE_PID:-}" "${LISTEN_PID:-}" 2>/dev/null || true
  wait 2>/dev/null || true
  echo "Done."
}
trap cleanup EXIT INT TERM

echo "=== Starting host application ==="
sparkplugbctl host --broker "${BROKER}" --host-id scada-01 &
HOST_PID=$!
sleep 1

echo
echo "=== Starting listener ==="
sparkplugbctl listen --broker "${BROKER}" &
LISTEN_PID=$!
sleep 1

echo
echo "=== Starting edge node ==="
sparkplugbctl edge \
  --broker "${BROKER}" \
  --group plant \
  --node gateway-01 \
  --device motor1 \
  --metric "temperature:Double=22.5" \
  --metric "rpm:UInt32=1500" \
  --interval 5s &
EDGE_PID=$!

echo
echo "All processes running. Press Ctrl+C to stop."
wait
