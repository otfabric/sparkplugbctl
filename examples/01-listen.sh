#!/usr/bin/env bash
# Listen to all Sparkplug B traffic on a local broker.
# Run in one terminal while other examples run in another.
set -euo pipefail

BROKER="${BROKER:-tcp://localhost:1883}"

echo "Listening for Sparkplug B traffic on ${BROKER}..."
echo "Press Ctrl+C to stop."
echo

sparkplugbctl listen --broker "${BROKER}"
