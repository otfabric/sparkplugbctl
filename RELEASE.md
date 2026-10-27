# sparkplugbctl Releases

## v0.1.0

**Date:** 2026-10-27
**Previous release:** none

### Summary

The first release: a Sparkplug B 3.0.0 command-line tool for monitoring,
publishing, simulating and debugging MQTT infrastructure, built on
[go-sparkplugb](https://github.com/otfabric/go-sparkplugb) v0.1.1. That
library is qualified against six independent Sparkplug stacks on four
MQTT brokers through the
[sparkplugb-interop v0.1.0](https://github.com/otfabric/sparkplugb-interop/releases/tag/v0.1.0)
images.

### Highlights

- **`listen`:** Subscribes to `spBv1.0/#` and decodes every Sparkplug message live, a primary host's STATE (JSON) included; filters by group, node and device.
- **`publish`:** Publishes one Sparkplug B payload with typed metrics (`name:Type=value`).
- **`edge`:** Simulates an edge node: NBIRTH and DBIRTH, periodic data, the NCMD and DCMD it receives, rebirth on request. With `--primary-host` it waits for the host's STATE online before its births, and when the host goes offline publishes NDEATH, reconnects and waits again.
- **`host`:** Runs a primary host application: STATE, and the births, data and deaths of every edge node.
- **`app`:** Runs an application that tracks the edge nodes without publishing STATE.
- **`decode` and `encode`:** Convert between typed metrics and the binary Sparkplug B protobuf, on stdin and stdout or with `--file`.
- **Datatypes:** `--metric` takes the integer, float, Boolean, string, UUID and bytes scalars; `listen` and `decode` show every datatype, `DateTime`, arrays, `DataSet` and `Template` included.
- **Connection:** `--broker` URL, username and password, TLS client certificate and CA (`--tls-cert`, `--tls-key`, `--tls-ca`); every global flag also as a `SPARKPLUGBCTL_*` variable (`SPARKPLUGBCTL_BROKER`, `SPARKPLUGBCTL_TLS_CA`, …), the command line taking precedence.
- **Output:** Text, or JSON with `--json`; shell completion for bash, zsh, fish and PowerShell.
- **`version`:** Prints the go-sparkplugb version the binary is built with.

### Limitations

- MQTT 3.1.1 only, as go-sparkplugb.
- `--metric` does not take `DateTime`, arrays, `DataSet` or `Template`.

---
