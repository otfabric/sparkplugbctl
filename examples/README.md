# sparkplugbctl Examples

Runnable shell scripts demonstrating each sparkplugbctl command. All examples
assume a local MQTT broker on `tcp://localhost:1883` (override with `--broker`).

## Prerequisites

```bash
# Install sparkplugbctl
go install github.com/otfabric/sparkplugbctl@latest

# Start a local broker (e.g. Mosquitto)
docker run -d --name mosquitto -p 1883:1883 eclipse-mosquitto:2.0.21 \
  mosquitto -c /dev/null -p 1883
```

## Examples

| Script | Description |
|--------|-------------|
| [`01-listen.sh`](01-listen.sh) | Subscribe to all Sparkplug B traffic and decode payloads live |
| [`02-edge-node.sh`](02-edge-node.sh) | Simulate an edge node with custom metrics and devices |
| [`03-host-app.sh`](03-host-app.sh) | Run a host application that tracks all edge nodes |
| [`04-application.sh`](04-application.sh) | Run an application subscriber (no STATE) |
| [`05-encode-decode.sh`](05-encode-decode.sh) | Encode metrics to binary and decode back |
| [`06-publish.sh`](06-publish.sh) | Publish a single Sparkplug B payload |
| [`07-edge-with-host.sh`](07-edge-with-host.sh) | Full scenario: host + edge node + listener in parallel |
