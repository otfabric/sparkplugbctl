# sparkplugbctl Releases

## v0.6.0

**Date:** 2026-11-15
**Previous release:** v0.5.0

Logins for every command that connects, a password file, a bound on the payloads `listen` and `decode` decode, escaped text output, and checks of the command line before `edge` and `publish` connect. Adds `--password-file`, `SPARKPLUGBCTL_PASSWORD_FILE` and `--max-payload`; `--password` is deprecated. Some exit codes change (below); the JSON output is unchanged.

### Added

- `--password-file` and `SPARKPLUGBCTL_PASSWORD_FILE`: the MQTT password from a file, without one trailing `\n` or `\r\n`, at most the 65535 bytes MQTT allows. Only the commands that connect read it. A missing or empty file, or a longer one, is exit 3, one the operating system does not let it read exit 9; `--password` together with `--password-file`, or both variables, is exit 2. `--password-file` on the command line takes precedence over `SPARKPLUGBCTL_PASSWORD`, and `--password` over `SPARKPLUGBCTL_PASSWORD_FILE`.
- `--max-payload` of `listen` and `decode`: the largest payload they decode, in bytes; default 16777216 (16 MiB), far above what a Sparkplug payload carries in practice.

### Breaking

Exit codes that scripts see differently (AGENTS.md hard rule 1; docs/exitcodes.md):

- A failed write to stdout is exit 10, "the output could not be written": for `decode`, `encode`, `publish` and `version` (before: 0, and 3 for `encode`), and for `listen`, `host` and `app` at the first message they cannot write, after they disconnect (before, they ran on and exited 0). No way back: the output was lost before; a script that ignored that can accept 10.
- `decode` of an input above `--max-payload`: exit 3; it was decoded before. `--max-payload` raises the bound (up to `math.MaxInt-1`).
- `edge --interval` of 0 or less: exit 3, before connecting (before, the command stopped with a runtime error, exit 2, after connecting). Give a positive `--interval`.
- `edge` with a `--metric` that does not parse: exit 3 as before, now before connecting, so the broker publishes no NDEATH of the node. None needed: the exit code is the same, and comes earlier.
- `publish --topic` with `+` or `#`: exit 3, before connecting (before, exit 0; the topic of an MQTT PUBLISH has no wildcards, and the message reached no subscriber). None needed: such a message reached no subscriber.
- `publish --seq` above 255: exit 3, before connecting (before, published with that number; a Sparkplug sequence number is 0 to 255). Use a sequence number from 0 to 255.

### Changed

- `edge`, `host` and `app` log in with `--username` and the password (`--password`, `--password-file` or their variables), as `listen` and `publish` do; they connected without a login before.
- The help text marks `--password` deprecated, in favour of `--password-file` or `SPARKPLUGBCTL_PASSWORD`. It still works.
- A password without `--username` is not sent by any command, and a warning on stderr says so: MQTT carries a password only with a username. `listen` and `publish` left it out without a word before.
- A `--tls-cert`, `--tls-key` or `--tls-ca` with a broker URL whose scheme connects without TLS (`tcp://`, `mqtt://`, `ws://`, `unix://`, or no scheme) prints a warning on stderr naming the flag and the scheme. The command connects without TLS, as before.
- `listen`, `host`, `app` and `edge` print the broker URL with the password of its user information replaced by `xxxxx`.
- Text output escapes the control characters, bidirectional formatting characters, line and paragraph separators and bytes that are not UTF-8 of what comes from the broker, as Go escapes (`\x1b`, `\u202e`, `\xff`): the topic, metric names, string values and STATE host ID of `listen` and `decode`, the event lines of `host` and `app`, the decode-error lines of `listen` and the DCMD lines of `edge`. Names without such characters print as before; the JSON output is unchanged.
- `listen` and `decode` format and write the metrics of a message one at a time, instead of holding the whole output until the end. The JSON output is the same, byte for byte. In text, the metric lines are aligned in blocks of up to 4096 lines or 256 KiB; before, all the lines of a message were aligned together.
- `listen` reports a message above `--max-payload` on stderr, `decode error on <topic>: payload of N bytes is above the limit of M bytes (--max-payload)`, and goes on.
- `edge` checks `--metric` and `--interval` before it connects.
- `edge` prints `Published DDEATH for <device>` only for a DDEATH that was published, and `publish DDEATH <device>: <error>` for one that was not.

### Documentation

- README.md: the new flags and variable, logins, TLS schemes, the payload bound and the escaped text output; `make lint` runs staticcheck only (it said `go vet` too).
- `app --help` no longer says that the command can send NCMD/DCMD commands: it has no option for that.
- SECURITY.md: TLS schemes, credential handling and the password file, the payload bound and memory, escaped text output.
- docs/exitcodes.md: the cases above.

### Tests

- `TestCredentials`: `listen`, `publish`, `edge`, `host` and `app` against a broker that admits one login, with the flags, the variables and the password file; a wrong password and no login exit 4; the password is in no output. `TestPasswordWithoutUsername`, `TestPasswordFile`, `TestReadPasswordBounds`, `TestPasswordHelp`.
- `TestTLSFlagsWithoutTLSScheme` and `TestTLSFlagsWarningOnCommand`; `TestRedactURL` and `TestBrokerURLPasswordNotPrinted`.
- `TestDecodeMaxPayload`, `TestDecodeLargeInput`, `TestPayloadLimit` (the bound on 32-bit platforms) and `TestListenMaxPayload`. In `internal/output`, `TestPrintMemoryBudget` bounds the heap held while a payload of 16384 array metrics is printed, in text and JSON; `TestPrintJSONSameDocument`, `TestPrintTextSameColumns` and `TestPrintTextBlocks` compare the output with the form of v0.5.0.
- `TestEscape`, `TestPrintTextEscapes` and `TestTextOutputEscapesBrokerStrings`.
- `TestOutputWriteErrors`, `TestStreamWriteErrors`, `TestPrintWriteError` and `TestEdgeDeviceDeathFailure`.
- `TestEdgeValidatesBeforeConnect` and `TestPublishValidatesBeforeConnect`.
- `TestEdgePrimaryHost` drives the publish interval and records the transport, instead of waiting 100 ms: nothing is published over three intervals before the primary host's STATE, then NBIRTH, DBIRTH and NDATA in that order. `TestEdgePrimaryHostBroker` keeps the run through a broker.

---

## v0.5.0

**Date:** 2026-11-15
**Previous release:** v0.4.8

Moves to go-sparkplugb v0.6.0 (was v0.5.0). No command, flag or exit code is added or changed.

### Changed

- Built on go-sparkplugb v0.6.0 (go-sparkplugb RELEASE.md, v0.6.0). For the tool this means:
  - `host` and `app` no longer print an NDEATH of an edge node they have not seen born (no NBIRTH since they connected), in text or JSON output.
  - The log lines of `host` and `app` on stderr quote the group, edge node and device IDs, e.g. `NBIRTH from "g1/n1"`.
  - `edge` publishes its NBIRTH and DBIRTHs on each connection once the node is online, as before; an NCMD asking for a rebirth before that NBIRTH is logged and not acted on.
  - `listen` and `decode` leave out the value of a metric of an array datatype above 1 << 20 elements (no `value` in JSON, `<nil>` in text), as for any value the library does not decode; before, only BooleanArray and StringArray metrics were limited.
  - `version` lists `go-sparkplugb 0.6.0`.

---

## v0.4.8

**Date:** 2026-11-15
**Previous release:** v0.4.7

No command or flag is added or changed.

### Changed

- Built on go-sparkplugb v0.5.0 (was v0.4.3) (go-sparkplugb RELEASE.md, v0.5.0). The release adds session checks that a host or application switches on; the tool does not set them, so its behaviour is unchanged.

---

## v0.4.7

**Date:** 2026-11-14
**Previous release:** v0.4.6

Moves to go-sparkplugb v0.4.3 (was v0.4.2). The commands' options and output are unchanged.

### Changed

- Built on go-sparkplugb v0.4.3 (go-sparkplugb RELEASE.md, v0.4.3). For the tool this means:
  - A metric, property, template parameter or DataSet cell whose value is carried in another field than its datatype's is reported with the library's type-mismatch error, where a value that is not an integer was shown as zero.
  - Templates nested more than 32 deep are refused with the library's invalid-template error.
  - The host commands hold at most 4096 messages and 8 MiB while they wait for a missing sequence number, over all edge nodes, and a held message waits at most 30 s.
  - A group, edge node or host ID that cannot be a topic segment (`/`, `+`, `#`, or not valid UTF-8) is refused when the client is created.

---

## v0.4.6

**Date:** 2026-11-13
**Previous release:** v0.4.5

Exit codes and the JSON fields are unchanged.

### Changed

- go-sparkplugb v0.4.2 (was v0.4.1). In `edge`, `host` and `app`, a `Connect` that fails after the broker's CONNACK — a subscription, or the STATE birth of `host` — now closes the connection with a DISCONNECT instead of leaving it up with its will armed until the command exits; `host` first publishes its STATE death when its STATE birth may have reached the broker. The command still exits with the `connect:` error and exit code it had. The library's default client ID of an application is now `sparkplug_app_` and 8 hex digits from `crypto/rand` (was the clock modulo 100000); `sparkplugbctl app` sets its own, below. `version` lists `go-sparkplugb 0.4.2`.

### Fixed

- Without `--client-id`, `listen` connected as `sparkplugbctl-listen` and `publish` as `sparkplugbctl-publish`, so two of them at once against one broker disconnected each other (a broker drops a client when another connects with its client ID, MQTT-3.1.4-2). `app` used the clock modulo 100000, at most 100000 IDs and 100 on a clock of microseconds. `app`, `listen` and `publish` now connect as `sparkplugbctl-<command>-` and 8 random hex digits from `crypto/rand`, a new one each run, e.g. `sparkplugbctl-listen-3f9a0c1e`. `host` (`sparkplugbctl-host-<host-id>`) and `edge` (`sparkplugbctl-edge-<group>-<node>`) keep the IDs of their Sparkplug identity. A `--client-id` or `SPARKPLUGBCTL_CLIENT_ID` is used as given. The `--client-id` help and README.md describe the defaults. The new defaults are 26 to 30 bytes: MQTT 3.1.1 obliges a broker to accept 1 to 23 and lets it accept more (mochi-mqtt, which the tests run against, does); for a broker that holds to 23, `--client-id` sets a shorter one, as for the `host` and `edge` IDs.

### Tests

- `TestRandomClientID`: the format of the default of `app`, `listen` and `publish` and no repeat in 64 draws.
- `TestDefaultClientIDs`: the client ID each command connects with: a random ID of that format for `app`, `listen` and `publish`, different in two runs; the fixed ID of `host` and `edge`; `--client-id` and `SPARKPLUGBCTL_CLIENT_ID` as given.
- `TestConnectAfterConnackFaults`: a failed subscription of `app`, `host` and `edge` (with and without `--primary-host`), and a failed STATE birth of `host`, exit with the `connect:` error and code 4 and close the connection; on go-sparkplugb v0.4.1 the connection was left open.

---

## v0.4.5

**Date:** 2026-11-13
**Previous release:** v0.4.4

SAST / Go 1.26 hardening release: go-sparkplugb v0.4.1, a static analysis pass and otfabric/engineering policy 1.5. Exit codes and the JSON fields are unchanged.

### Changed

- go-sparkplugb v0.4.1 (was v0.4.0): the element count of a received BooleanArray is held to the library's limit on 32-bit platforms (`linux/arm/v7`) too; `version` lists `go-sparkplugb 0.4.1`.
- `encode --file` creates the payload file with mode `0600` (was `0644`), as the other OT Fabric tools do for the files they write.
- otfabric/engineering policy 1.5 (was 1.4): `.otfabric/policy.yaml`, the policy checker v1.5.0 in CI and the docs check, and the shared workflows v3.6.0. `otfabric-policy check` v1.5.0: compliant, no failing rule.

### Fixed

- A message, metric or STATE timestamp past 2^63-1 milliseconds was printed as a date before 1970; it is now printed as its number of milliseconds (docs/json-output.md).

### Security

- SAST review of sparkplugbctl (gosec, opengrep at severity and confidence low): 10 findings, 7 fixed in code (the timestamp conversions above, the file mode of `encode --file`, a `defer` in the publish loop of `edge` that ran once and is now a plain call) and 3 recorded as false positives in a private triage repository, not in this one (the datatype of a metric, which an unknown value cannot alias; the file the operator names for `decode --file`; `math/rand/v2` for the simulated values of `edge`). There is no `nolint` or `#nosec`.
- SECURITY.md has a Static Analysis section.

### Tests

- `TestTimestampOutOfRange`: payload, metric and STATE timestamps of 2^64-1 ms, as text and JSON, and the formatting at 2^63-1 ms.
- `TestEncodeDecodeFile` checks that the file `encode --file` writes has no group or other access.

---

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
