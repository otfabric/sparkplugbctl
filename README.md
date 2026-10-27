# sparkplugbctl

Command-line tool for Sparkplug B 3.0 MQTT infrastructure.

## Install

```bash
go install github.com/otfabric/sparkplugbctl@latest
```

Or build from source:

```bash
make build
# binary at bin/sparkplugbctl
```

## Commands

### listen

Subscribe to Sparkplug B messages and print them in real time.

```bash
# Listen to all messages
sparkplugbctl listen --broker tcp://localhost:1883

# Filter by group and node
sparkplugbctl listen --group plant --node gateway

# JSON output
sparkplugbctl listen --json
```

Output:

```
[NBIRTH] spBv1.0/plant/NBIRTH/gateway
  seq=0  ts=2026-10-27T12:00:00Z
  bdSeq          Int64    1
  temperature    Double   22.5    quality=GOOD
  humidity       Float    55.0
```

### edge

Simulate a Sparkplug B edge node with automatic data publishing.

```bash
# Default metrics (temperature, humidity, pressure)
sparkplugbctl edge --group plant --node gateway

# Custom metrics and devices
sparkplugbctl edge --group plant --node gw1 \
  --device sensor1 --device sensor2 \
  --metric "temperature:Double=22.5" \
  --metric "pressure:Double=1013.25" \
  --interval 2s

# With primary host
sparkplugbctl edge --group plant --node gw1 --primary-host scada
```

### host

Simulate a Sparkplug B host application.

```bash
sparkplugbctl host --host-id scada --broker tcp://localhost:1883
```

### publish

Publish a single Sparkplug B payload.

```bash
sparkplugbctl publish \
  --topic "spBv1.0/plant/NDATA/gateway" \
  --metric "temperature:Double=23.1" \
  --metric "status:Boolean=true" \
  --seq 5
```

### decode

Decode a binary Sparkplug B Protobuf payload.

```bash
# From stdin
cat payload.bin | sparkplugbctl decode

# From file
sparkplugbctl decode --file payload.bin

# JSON output
sparkplugbctl decode --file payload.bin --json
```

### encode

Encode metrics into a binary Sparkplug B Protobuf payload.

```bash
# To stdout (pipe to file or another tool)
sparkplugbctl encode \
  --metric "temperature:Double=22.5" \
  --metric "status:Boolean=true" \
  > payload.bin

# To file
sparkplugbctl encode --file payload.bin \
  --metric "temperature:Double=22.5"
```

### version

```bash
sparkplugbctl version
```

## Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--broker` | `tcp://localhost:1883` | MQTT broker URL |
| `--client-id` | auto-generated | MQTT client ID |
| `--username` | | MQTT username |
| `--password` | | MQTT password |
| `--tls-cert` | | TLS client certificate |
| `--tls-key` | | TLS client key |
| `--tls-ca` | | TLS CA certificate |
| `--json` | `false` | Output as JSON |

## Supported Metric Types

`Int8`, `Int16`, `Int32`, `Int64`, `UInt8`, `UInt16`, `UInt32`, `UInt64`,
`Float`, `Double`, `Boolean`, `String`, `Text`, `UUID`, `Bytes`

## License

MIT — see [LICENSE](LICENSE).
