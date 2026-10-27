#!/usr/bin/env bash
# Publish a single Sparkplug B payload to a specific topic.
set -euo pipefail

BROKER="${BROKER:-tcp://localhost:1883}"

echo "Publishing NDATA payload to spBv1.0/plant/NDATA/gateway-01..."
echo

sparkplugbctl publish \
  --broker "${BROKER}" \
  --topic "spBv1.0/plant/NDATA/gateway-01" \
  --metric "temperature:Double=23.1" \
  --metric "status:Boolean=true" \
  --seq 5

echo "Done."
