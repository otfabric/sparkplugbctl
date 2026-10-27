#!/usr/bin/env bash
# Run a Sparkplug B host application that publishes STATE and tracks edge nodes.
set -euo pipefail

BROKER="${BROKER:-tcp://localhost:1883}"

echo "Starting host application (host-id: scada-01)..."
echo "Will publish STATE birth/death and track all edge nodes."
echo

sparkplugbctl host \
  --broker "${BROKER}" \
  --host-id scada-01
