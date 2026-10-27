#!/usr/bin/env bash
# Simulate a Sparkplug B edge node with two devices and custom metrics.
set -euo pipefail

BROKER="${BROKER:-tcp://localhost:1883}"

echo "Starting edge node (plant/gateway-01) with 2 devices..."
echo

sparkplugbctl edge \
  --broker "${BROKER}" \
  --group plant \
  --node gateway-01 \
  --device sensor1 \
  --device motor1 \
  --metric "temperature:Double=22.5" \
  --metric "pressure:Double=1013.25" \
  --metric "status:Boolean=true" \
  --interval 3s
