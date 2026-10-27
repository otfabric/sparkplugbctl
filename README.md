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
base64 -d < payload.b64 | sparkplugbctl decode

# Run a host application that tracks every edge node
sparkplugbctl host --host-id scada --broker tcp://localhost:1883

# Run a passive application (no STATE, just tracking)
sparkplugbctl app --broker tcp://localhost:1883
```

---

## Commands

```
sparkplugbctl
├── listen         Subscribe to spBv1.0/# and decode Sparkplug payloads and STATE live
├── publish        Publish a single Sparkplug B payload with typed metrics
├── edge           Simulate a Sparkplug B edge node (birth, data, rebirth)
├── host           Simulate a Sparkplug B host application (STATE, tracking)
├── app            Run a Sparkplug B application (subscriber, no STATE)
├── decode         Decode a binary Sparkplug B Protobuf payload (stdin or file)
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

Subscribe to all Sparkplug B topics and decode payloads in real time. A
primary host's STATE message (JSON) is shown as such.

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
[STATE] spBv1.0/STATE/scada
  online=true  ts=2026-10-27T11:12:18Z

[NBIRTH] spBv1.0/plant/NBIRTH/gw1
  seq=0  ts=2026-10-27T11:12:18Z
  bdSeq                 Int64    1
  Node Control/Rebirth  Boolean  false
  temperature           Double   22.5

[DBIRTH] spBv1.0/plant/DBIRTH/gw1/sensor1
  seq=1  ts=2026-10-27T11:12:18Z
  status  Boolean  true

[NDATA] spBv1.0/plant/NDATA/gw1
  seq=2  ts=2026-10-27T11:12:20Z
  temperature  Double  22.816084857923634
```

A metric with a quality property shows it, e.g. `quality=STALE`.

### edge

Simulate a Sparkplug B edge node with automatic lifecycle management and
periodic data publishing. Each device is born with one metric, `status`
(Boolean). Received NCMD and DCMD messages are reported; a rebirth request
is answered with new births.

With `--primary-host`, the node waits for that host's STATE to be online
before its births. When the host goes offline, the node publishes NDEATH,
reconnects and waits again, as Sparkplug 3.0.0 requires.

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
[INFO]  state: disconnected -> connecting
[INFO]  state: connecting -> subscribing
[INFO]  state: subscribing -> waiting_for_primary_host
Connected to tcp://localhost:1883 as plant/gw1
Waiting for primary host scada to be online
Publishing data every 2s (Ctrl+C to stop)
[INFO]  state: waiting_for_primary_host -> online
Published NBIRTH (1 metrics)
Published DBIRTH for sensor1
Published NDATA (1 metrics)
^C
Shutting down edge node...
Published DDEATH for sensor1
Disconnected
```

### host

Run a Sparkplug B host application that publishes STATE birth/death
certificates and tracks all edge nodes.

```bash
sparkplugbctl host --host-id scada --broker tcp://localhost:1883
```

Example output:

```
[INFO]  host state: disconnected -> connecting
[INFO]  host state: connecting -> connected
Host scada online (broker: tcp://localhost:1883)
Listening for edge activity (Ctrl+C to stop)
[NBIRTH] plant/gw1 (3 metrics)
[DBIRTH] plant/gw1/sensor1 (1 metrics)
[NDATA]  plant/gw1 (1 metrics)
^C
Shutting down host...
[INFO]  host state: connected -> disconnected
Disconnected
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
[INFO]  application state: disconnected -> connecting
[INFO]  application state: connecting -> connected
Application mode (no STATE) — broker: tcp://localhost:1883
Listening for edge activity (Ctrl+C to stop)
2026-10-27T12:12:18+01:00  [NBIRTH] plant/gw1 (3 metrics)
2026-10-27T12:12:18+01:00  [DBIRTH] plant/gw1/sensor1 (1 metrics)
2026-10-27T12:12:20+01:00  [NDATA]  plant/gw1 (1 metrics)
2026-10-27T12:12:30+01:00  [NDEATH] plant/gw1
^C
Shutting down application...
[INFO]  application state: connected -> disconnected
[INFO]  application state: disconnected -> closed
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

Decode a binary Sparkplug B Protobuf payload from stdin or a file. Convert
hex or base64 to binary first.

```bash
# From a file
sparkplugbctl decode --file payload.bin

# Binary on stdin
sparkplugbctl encode --metric "temperature:Double=23.1" | sparkplugbctl decode

# From hex or base64
xxd -r -p payload.hex | sparkplugbctl decode
base64 -d < payload.b64 | sparkplugbctl decode

# JSON output
sparkplugbctl decode --file payload.bin --json
```

Example output:

```
seq=5  ts=2026-10-27T11:09:22Z
  temperature  Double   23.1
  status       Boolean  true
```

### encode

Encode typed metrics into a binary Sparkplug B Protobuf payload.

```bash
# To stdout (pipe to file or another tool)
sparkplugbctl encode \
  --metric "temperature:Double=22.5" \
  --metric "status:Boolean=true" \
  > payload.bin

# To file, with a sequence number
sparkplugbctl encode --file payload.bin --seq 5 \
  --metric "temperature:Double=22.5"

# Base64 output
sparkplugbctl encode --metric "temperature:Double=22.5" | base64
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
sparkplugbctl v0.1.0
  commit:  abc1234
  built:   2026-10-27T02:00:00Z
  library: go-sparkplugb v0.1.2
  go:      go1.27.3
  os/arch: darwin/arm64
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

`--metric` (on `publish`, `encode` and `edge`) takes `Int8`, `Int16`,
`Int32`, `Int64`, `UInt8`, `UInt16`, `UInt32`, `UInt64`, `Float`, `Double`,
`Boolean` (or `Bool`), `String`, `Text`, `UUID` and `Bytes` (the value's
UTF-8 bytes); type names are case-insensitive.

Metric syntax: `name:Type=value` — for example `temperature:Double=22.5` or
`active:Boolean=true`.

`listen` and `decode` show metrics of every datatype a payload carries,
including `DateTime`, arrays, `DataSet` and `Template`.

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
| `SPARKPLUGBCTL_TLS_CERT` | `--tls-cert` | |
| `SPARKPLUGBCTL_TLS_KEY` | `--tls-key` | |
| `SPARKPLUGBCTL_TLS_CA` | `--tls-ca` | |
| `SPARKPLUGBCTL_JSON` | `--json` | `false` |

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
