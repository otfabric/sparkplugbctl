# sparkplugbctl Releases

## v0.4.4

**Date:** 2026-11-13
**Previous release:** v0.4.3

### Changed

- go-sparkplugb v0.4.0 (was v0.3.6). For `host` and `app`, a received template that breaks the Sparkplug 3.0.0 rules is printed as a `[WARN]` line on stderr: an instance of a definition that is not in the NBIRTH, a definition outside the NBIRTH or with a `template_ref`, a member or parameter the definition does not have, or a birth instance that lacks members. The message is still shown. The SparkplugNet 1.4.1 edge node, for one, sends its definitions with an empty `template_ref`, so each of its NBIRTHs prints such a warning. `edge` and `publish` send scalar metrics only, so the new publish-side template errors of go-sparkplugb cannot arise in them. The `sparkplugbctl` code is unchanged, and `version` lists `go-sparkplugb 0.4.0`.

---

## v0.4.3

**Date:** 2026-11-12
**Previous release:** v0.4.2

### Changed

- go-sparkplugb v0.3.6 (was v0.3.5): its interop suite pins sparkplugb-interop v0.4.0; the library does not change. `sparkplugbctl` code is unchanged; `version` lists `go-sparkplugb 0.3.6`.

---

## v0.4.2

**Date:** 2026-11-12
**Previous release:** v0.4.1

### Fixed

- Through go-sparkplugb v0.3.5 (was v0.3.4), for `edge`: an NDATA published while the node answers an NCMD rebirth no longer goes out after the new NBIRTH with a `seq` of the ending session, and no NDATA goes out between the NBIRTH and the DBIRTHs; it waits and follows them. A host took such a message for the new session's message with that `seq` and lost the latter (go-sparkplugb#24). An NBIRTH is also stamped in a later millisecond than the message before it, unless the clock went back.
- Through go-sparkplugb v0.3.5, for every command that publishes: a publish that gives up on its context or timeout (`publish` on Ctrl+C, the DDEATHs of `edge` at shutdown) can no longer be overtaken by a later message, and one whose context has already ended is not sent. `sparkplugbctl` code is unchanged; `version` lists `go-sparkplugb 0.3.5`.
- `make build` marks the commit `-dirty` when the working tree has changes, as `go build` does. Release binaries are unchanged: the release workflow sets the commit.

---

## v0.4.1

**Date:** 2026-11-08
**Previous release:** v0.4.0

### Changed

- Built on go-sparkplugb v0.3.4 (was v0.3.3). Its `edge` fix makes a rebirth (`Node.Rebirth`, and so the answer to an NCMD rebirth) publish the DBIRTHs in the order the devices were first born, instead of map order, and reads the devices' state under the lock, which removes a data race with a device death or data published at the same time. `version` lists `go-sparkplugb 0.3.4`.
- The minimum Go version is now 1.26.0 (was 1.27.3): `go.mod` declares `go 1.26.0`, and CI tests Go 1.26 and 1.27.
- Release binaries are built with the latest Go 1.26 patch release (otfabric/.github release workflow v3.6.0); they were built with the exact Go version in `go.mod`.

### Tests

- `TestVersion` passes on Go 1.26, whose test binaries record no dependencies, so the expected line has no libraries there; the e2e test checks that the built binary names go-sparkplugb in `version` and `version --json`.

---

## v0.4.0

**Date:** 2026-11-07
**Previous release:** v0.3.6

### Added

- `version --json` (and `version` with `SPARKPLUGBCTL_FORMAT=json`) prints one JSON object: `name`, `version`, `tag`, `commit`, `built`, `libraries`, `go`, `platform`, in that order ([docs/json-output.md](docs/json-output.md#version)).
- Shell completion for flag values: the type in `--metric name:type=value` (`publish`, `edge`, `encode`) completes the 15 metric types, `--tls-cert`/`--tls-ca` complete `.pem`/`.crt`/`.cer` files and `--tls-key` `.pem`/`.key` files; flags that take free values and commands without arguments no longer offer file names. `completion` covers bash, zsh, fish and PowerShell.

### Changed

- `version` prints one line, aligned with the other OT Fabric CLIs: `sparkplugbctl 0.4.0 (tag v0.4.0, commit <full SHA>, built <RFC 3339>, go-sparkplugb 0.3.3, go1.27.3, darwin/arm64)`. The version has no leading `v` and the tag part is left out when there is none. A `go install …@vX.Y.Z` build reports that version; a `go build` or a `make build` off a tag reports `dev` with the VCS commit (with `-dirty`).
- The release workflow injects the git tag (`-X main.tag`); the Makefile injects the full commit SHA, and the version and tag only when HEAD is exactly on a tag.

---

## v0.3.6

**Date:** 2026-11-01
**Previous release:** v0.3.5

### Fixed

- Through go-sparkplugb: an update in the millisecond of a rebirth is kept, and an edge node no longer publishes an NBIRTH after its NDEATH. `sparkplugbctl` code is unchanged.
- Built on go-sparkplugb v0.3.3.

---

## v0.3.5

**Date:** 2026-11-01
**Previous release:** v0.3.4

### Fixed

- An NBIRTH delivered just after the NDEATH of its `bdSeq` no longer brings a dead edge node back online, through go-sparkplugb. `sparkplugbctl` code is unchanged.
- Built on go-sparkplugb v0.3.2.

---

## v0.3.4

**Date:** 2026-11-01
**Previous release:** v0.3.3

### Fixed

- Built on go-sparkplugb v0.3.1, which keeps the sessions of an edge node apart when a broker reorders messages across topics. `sparkplugbctl` code is unchanged.
- Built on go-sparkplugb v0.3.1.

---

## v0.3.3

**Date:** 2026-11-01
**Previous release:** v0.3.2

### Changed

- Built on go-sparkplugb v0.3.0, which refuses every integer out of its datatype's range: a metric whose value does not fit its datatype, such as a UInt8 of 256, now shows no value in the output, where it used to show the truncated value.

---

## v0.3.2

**Date:** 2026-11-01
**Previous release:** v0.3.1

### Changed

- Built on go-sparkplugb v0.2.3, which refuses an Int8 or Int16 that is neither sign-extended nor width-sized: such a metric now shows no value in the output.
- Built on go-sparkplugb v0.2.3.

---

## v0.3.1

**Date:** 2026-10-31
**Previous release:** v0.3.0

### Fixed

- `decode --json` and `listen --json` printed nothing and exited 0 when a metric was NaN, infinite, an array holding one of those, or a DateTime past year 9999. Those values are written as text.
- Built on go-sparkplugb v0.2.1.

---

## v0.3.0

**Date:** 2026-10-30
**Previous release:** v0.2.0

### Dependencies

- `golang.org/x/net` → v0.62.0 (govulncheck).

---

## v0.2.0

**Date:** 2026-10-30
**Previous release:** v0.1.1

### Summary

Exit codes converge on the organisation table (ADR-0006). **Scripts that test for exit 1 must change.**

| Case | Old | New |
|---|---|---|
| Invalid flags or arguments | 1 | 2 |
| Unreadable input, payload that does not decode, invalid metric | 1 | 3 |
| Connecting, subscribing or publishing failed | 1 | 4 |
| Timeout | 1 | 5 |
| Permission refused by the operating system | 1 | 9 |
| Output or unexpected error | 1 | 10 |
| Interrupted before a result | 1 | 130 |

### Added

- `SPARKPLUGBCTL_FORMAT` (canonical) and `SPARKPLUGBCTL_OUTPUT_FORMAT`: `json` has the effect of `--json`.

### Fixed

- **`--tls-ca` (and `SPARKPLUGBCTL_TLS_CA`):** Was ignored unless `--tls-cert` and `--tls-key` were also given, so a broker with a private CA failed verification against the system roots. It now applies with or without a client certificate.

---

## v0.1.1

**Date:** 2026-10-27
**Previous release:** v0.1.0

### Summary

Built on [go-sparkplugb](https://github.com/otfabric/go-sparkplugb) v0.1.3:
`host` and `app` keep a device whose DBIRTH a broker delivers before its
node's NBIRTH after a restart (NanoMQ and HiveMQ CE do in some runs).

### Changes

- **`host` and `app` commands:** go-sparkplugb v0.1.3 holds a message behind the expected `seq` for the node's next session and applies it after the NBIRTH; before, the device stayed offline unless a rebirth came.
- **Dependencies:** `spf13/pflag` v1.0.10, `golang.org/x/net` v0.61.0, `golang.org/x/sync` v0.24.0.

---

## v0.1.0

**Date:** 2026-10-27
**Previous release:** none

### Summary

The first release: a Sparkplug B 3.0.0 command-line tool for monitoring,
publishing, simulating and debugging MQTT infrastructure, built on
[go-sparkplugb](https://github.com/otfabric/go-sparkplugb) v0.1.2. That
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
