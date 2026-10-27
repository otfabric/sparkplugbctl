#!/usr/bin/env bash
# Run a Sparkplug B application (subscriber only, no STATE).
# Use this when you want to observe the namespace without being a primary host.
set -euo pipefail

BROKER="${BROKER:-tcp://localhost:1883}"

echo "Starting application (no STATE)..."
echo "Subscribing to spBv1.0/# and tracking all edge nodes."
echo

sparkplugbctl app \
  --broker "${BROKER}"
