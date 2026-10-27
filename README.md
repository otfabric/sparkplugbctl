# sparkplugbctl

Sparkplug B 3.0.0 CLI for MQTT monitoring, publishing, simulation and debugging.

[![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/otfabric/sparkplugbctl/actions/workflows/ci.yaml/badge.svg)](https://github.com/otfabric/sparkplugbctl/actions/workflows/ci.yaml)
[![Release](https://img.shields.io/github/v/release/otfabric/sparkplugbctl?include_prereleases)](https://github.com/otfabric/sparkplugbctl/releases)

sparkplugbctl is a command-line tool for exploring, monitoring, publishing,
simulating and debugging [Eclipse Sparkplug B 3.0.0](https://www.eclipse.org/tahu/spec/sparkplug_spec.pdf)
MQTT infrastructure. Built in Go on
[go-sparkplugb](https://github.com/otfabric/go-sparkplugb).

---

## Quick start

```bash
# Listen to all Sparkplug B traffic on a broker
sparkplugbctl listen --broker tcp://localhost:1883

# Simulate an edge node publishing temperature every 2 s
sparkplugbctl edge --group plant --node gw1 \
  --metric "temperature:Double=22.5" --interval 2s

# Decode a base64-encoded Sparkplug payload
echo "CJkBEP..." | sparkplugbctl decode --base64

# Run a host application that tracks every edge node
sparkplugbctl host --host-id scada --broker tcp://localhost:1883

# Run a passive application (no STATE, just tracking)
sparkplugbctl app --broker tcp://localhost:1883
```

---

## Commands

```
sparkplugbctl
├── listen         Subscribe to spBv1.0/# and decode Sparkplug payloads live
├── publish        Publish a single Sparkplug B payload with typed metrics
├── edge           Simulate a Sparkplug B edge node (birth, data, rebirth)
├── host           Simulate a Sparkplug B host application (STATE, tracking)
├── app            Run a Sparkplug B application (subscriber, no STATE)
├── decode         Decode a binary Sparkplug B Protobuf payload (hex, base64, file)
├── encode         Encode typed metrics into binary Sparkplug B Protobuf
├── completion     Generate shell completion (bash|zsh|fish|powershell)
└── version        Print version and build info
```

---

## Installation

### Download a prebuilt binary

| Platform       | Architecture | Archive                                          |
|----------------|:------------:|--------------------------------------------------|
| Linux          | amd64        | `sparkplugbctl_<version>_linux_amd64.tar.gz`     |
| Linux          | arm64        | `sparkplugbctl_<version>_linux_arm64.tar.gz`     |
| macOS          | arm64 (M1+)  | `sparkplugbctl_<version>_darwin_arm64.tar.gz`    |
| macOS          | amd64        | `sparkplugbctl_<version>_darwin_amd64.tar.gz`    |
| Windows        | amd64        | `sparkplugbctl_<version>_windows_amd64.zip`      |
| Windows        | arm64        | `sparkplugbctl_<version>_windows_arm64.zip`      |

Download the latest archive from
[GitHub Releases](https://github.com/otfabric/sparkplugbctl/releases), verify
the SHA-256 checksum, and place the binary on your `PATH`.

**Linux (amd64)**

```bash
VERSION=0.1.0
curl -fsSL "https://github.com/otfabric/sparkplugbctl/releases/download/v${VERSION}/sparkplugbctl_${VERSION}_linux_amd64.tar.gz" \
  | tar xz -C /usr/local/bin sparkplugbctl
sparkplugbctl version
```

**macOS (Apple Silicon)**

```bash
VERSION=0.1.0
curl -fsSL "https://github.com/otfabric/sparkplugbctl/releases/download/v${VERSION}/sparkplugbctl_${VERSION}_darwin_arm64.tar.gz" \
  | tar xz -C /usr/local/bin sparkplugbctl
sparkplugbctl version
```

**Windows (PowerShell)**

```powershell
$VERSION = "0.1.0"
Invoke-WebRequest `
  "https://github.com/otfabric/sparkplugbctl/releases/download/v$VERSION/sparkplugbctl_${VERSION}_windows_amd64.zip" `
  -OutFile sparkplugbctl.zip
Expand-Archive sparkplugbctl.zip -DestinationPath C:\Tools
$env:PATH += ";C:\Tools"
sparkplugbctl version
```

### Install with Go

```bash
go install github.com/otfabric/sparkplugbctl@latest
```

Requires Go 1.27 or later.

### Build from source

```bash
git clone https://github.com/otfabric/sparkplugbctl.git
cd sparkplugbctl
make build
# binary at bin/sparkplugbctl
```

### Update

Replace the binary with a newer release archive, or re-run `go install`.

---

## Global flags

| Flag | Default | Description |
|------|---------|-------------|
| `--broker` | `tcp://localhost:1883` | MQTT broker URL |
| `--client-id` | auto-generated | MQTT client ID |
| `--username` | | MQTT username |
| `--password` | | MQTT password |
| `--tls-cert` | | TLS client certificate file |
| `--tls-key` | | TLS client key file |
| `--tls-ca` | | TLS CA certificate file |
| `--json` | `false` | Output as JSON |

---

## Usage

### listen

Subscribe to all Sparkplug B topics and decode payloads in real time.

```bash
# All traffic
sparkplugbctl listen --broker tcp://localhost:1883

# Filter by group and node
sparkplugbctl listen --group plant --node gateway

# JSON output for scripting
sparkplugbctl listen --json
```

Example output:

```
[NBIRTH] spBv1.0/plant/NBIRTH/gateway
  seq=0  ts=2026-10-27T12:00:00Z
  bdSeq          Int64    1
  temperature    Double   22.5    quality=GOOD
  humidity       Float    55.0

[NDATA] spBv1.0/plant/NDATA/gateway
  seq=1  ts=2026-10-27T12:00:05Z
  temperature    Double   23.1
```

### edge

Simulate a Sparkplug B edge node with automatic lifecycle management and
periodic data publishing.

```bash
# Default metrics (temperature, humidity, pressure)
sparkplugbctl edge --group plant --node gateway

# Custom metrics and devices
sparkplugbctl edge --group plant --node gw1 \
  --device sensor1 --device sensor2 \
  --metric "temperature:Double=22.5" \
  --metric "pressure:Double=1013.25" \
  --interval 2s

# With primary host awareness
sparkplugbctl edge --group plant --node gw1 --primary-host scada
```

Example output:

```
2026-10-27T12:00:00Z  state: disconnected -> connecting
2026-10-27T12:00:00Z  state: connecting -> connected
2026-10-27T12:00:00Z  NBIRTH published  seq=0  metrics=3
2026-10-27T12:00:00Z  DBIRTH published  device=sensor1  seq=1  metrics=2
2026-10-27T12:00:02Z  NDATA  published  seq=2  temperature=22.6
2026-10-27T12:00:04Z  NDATA  published  seq=3  temperature=22.7
^C
2026-10-27T12:00:05Z  shutting down...
```

### host

Run a Sparkplug B host application that publishes STATE birth/death
certificates and tracks all edge nodes.

```bash
sparkplugbctl host --host-id scada --broker tcp://localhost:1883
```

Example output:

```
2026-10-27T12:00:00Z  STATE online   host=scada
2026-10-27T12:00:01Z  NBIRTH         plant/gateway        metrics=5
2026-10-27T12:00:01Z  DBIRTH         plant/gateway/motor1  metrics=2
2026-10-27T12:00:06Z  NDATA          plant/gateway        seq=1  metrics=1
2026-10-27T12:00:30Z  NDEATH         plant/gateway        bdSeq=0
2026-10-27T12:00:30Z  STATE offline  host=scada
```

### app

Run a Sparkplug B application that subscribes to all Sparkplug B traffic and
tracks edge nodes and devices — without publishing STATE birth/death
certificates. Use this when you want passive monitoring or when another tool is
the primary host.

```bash
# Monitor all edge activity
sparkplugbctl app --broker tcp://localhost:1883

# Filter by group
sparkplugbctl app --group plant

# JSON output for scripting
sparkplugbctl app --json
```

Example output:

```
Application mode (no STATE) — broker: tcp://localhost:1883
Listening for edge activity (Ctrl+C to stop)
2026-10-27T12:00:01+00:00  [NBIRTH] plant/gateway (5 metrics)
2026-10-27T12:00:01+00:00  [DBIRTH] plant/gateway/motor1 (2 metrics)
2026-10-27T12:00:06+00:00  [NDATA]  plant/gateway (1 metrics)
2026-10-27T12:00:30+00:00  [NDEATH] plant/gateway
^C
Shutting down application...
Disconnected
```

### publish

Publish a single Sparkplug B payload with typed metrics.

```bash
sparkplugbctl publish \
  --topic "spBv1.0/plant/NDATA/gateway" \
  --metric "temperature:Double=23.1" \
  --metric "status:Boolean=true" \
  --seq 5
```

### decode

Decode a binary Sparkplug B Protobuf payload from hex, base64, or a file.

```bash
# From hex on stdin
echo "0a0b74656d7065726174757265" | sparkplugbctl decode

# From base64
echo "CJkBEP..." | sparkplugbctl decode --base64

# From file
sparkplugbctl decode --file payload.bin

# JSON output
sparkplugbctl decode --file payload.bin --json
```

Example output:

```
Sparkplug B Payload
  seq=5  ts=2026-10-27T12:00:00Z
  temperature    Double   23.1
  status         Boolean  true
  humidity       Float    55.0    quality=GOOD
```

### encode

Encode typed metrics into a binary Sparkplug B Protobuf payload.

```bash
# To stdout (pipe to file or another tool)
sparkplugbctl encode \
  --metric "temperature:Double=22.5" \
  --metric "status:Boolean=true" \
  > payload.bin

# To file
sparkplugbctl encode --file payload.bin \
  --metric "temperature:Double=22.5"

# Base64 output
sparkplugbctl encode --base64 \
  --metric "temperature:Double=22.5"
```

### completion

Generate shell completion scripts.

```bash
# Bash
sparkplugbctl completion bash > /etc/bash_completion.d/sparkplugbctl

# Zsh
sparkplugbctl completion zsh > "${fpath[1]}/_sparkplugbctl"

# Fish
sparkplugbctl completion fish > ~/.config/fish/completions/sparkplugbctl.fish

# PowerShell
sparkplugbctl completion powershell > sparkplugbctl.ps1
```

### version

```bash
sparkplugbctl version
```

```
sparkplugbctl v0.1.0 (abc1234 2026-10-27T02:00:00Z)
  go-sparkplugb v0.0.0-20261027040647-4359bcc1f978
  go1.27.3 darwin/arm64
```

---

## Examples

Runnable shell scripts are in the [`examples/`](examples/) directory:

| Script | Description |
|--------|-------------|
| [`01-listen.sh`](examples/01-listen.sh) | Subscribe to all Sparkplug B traffic |
| [`02-edge-node.sh`](examples/02-edge-node.sh) | Simulate an edge node with devices and metrics |
| [`03-host-app.sh`](examples/03-host-app.sh) | Run a host application (STATE + tracking) |
| [`04-application.sh`](examples/04-application.sh) | Run an application subscriber (no STATE) |
| [`05-encode-decode.sh`](examples/05-encode-decode.sh) | Encode metrics to binary and decode back |
| [`06-publish.sh`](examples/06-publish.sh) | Publish a single Sparkplug B payload |
| [`07-edge-with-host.sh`](examples/07-edge-with-host.sh) | Full scenario: host + edge + listener |

```bash
# Run any example (start a local broker first)
./examples/02-edge-node.sh
```

---

## Supported metric types

`Int8`, `Int16`, `Int32`, `Int64`, `UInt8`, `UInt16`, `UInt32`, `UInt64`,
`Float`, `Double`, `Boolean`, `String`, `Text`, `UUID`, `Bytes`, `DateTime`,
`DataSet`, `Template`

Metric syntax: `name:Type=value` — for example `temperature:Double=22.5` or
`active:Boolean=true`.

---

## Environment variables

Every global flag can be set via an environment variable. The flag takes
precedence when both are set.

| Variable | Flag | Default |
|----------|------|---------|
| `SPARKPLUGBCTL_BROKER` | `--broker` | `tcp://localhost:1883` |
| `SPARKPLUGBCTL_CLIENT_ID` | `--client-id` | auto-generated |
| `SPARKPLUGBCTL_USERNAME` | `--username` | |
| `SPARKPLUGBCTL_PASSWORD` | `--password` | |

---

## Building and development

| Target | Description |
|--------|-------------|
| `make build` | Build the binary to `bin/sparkplugbctl` |
| `make test` | Run all tests |
| `make lint` | Run `go vet` and `staticcheck` |
| `make install` | Install to `$GOPATH/bin` |
| `make clean` | Remove build artifacts |

```bash
git clone https://github.com/otfabric/sparkplugbctl.git
cd sparkplugbctl
make build
make test
```

---

## License

MIT — see [LICENSE](LICENSE).
