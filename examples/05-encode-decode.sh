#!/usr/bin/env bash
# Encode Sparkplug B metrics to binary and decode them back.
set -euo pipefail

echo "=== Encode metrics to base64 ==="
ENCODED=$(sparkplugbctl encode --base64 \
  --metric "temperature:Double=22.5" \
  --metric "pressure:Double=1013.25" \
  --metric "active:Boolean=true")
echo "${ENCODED}"
echo

echo "=== Decode the payload ==="
echo "${ENCODED}" | sparkplugbctl decode --base64
echo

echo "=== Encode to file and decode with JSON output ==="
sparkplugbctl encode --file /tmp/sparkplug_payload.bin \
  --metric "rpm:UInt32=1500" \
  --metric "voltage:Float=230.5"

sparkplugbctl decode --file /tmp/sparkplug_payload.bin --json
rm -f /tmp/sparkplug_payload.bin
