# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in `sparkplugbctl`, please report it responsibly.

**Do not open a public GitHub issue for security vulnerabilities.**

Instead, please email **security@otfabric.io**, or open a private advisory on GitHub (the [Security](https://github.com/otfabric/sparkplugbctl/security) tab → **Advisories** → **Report a vulnerability**), with:

- A description of the vulnerability
- Steps to reproduce the issue, ideally the exact command
- The affected version(s) (`sparkplugbctl version` output)
- Any potential impact you've identified

We will acknowledge your report within 48 hours and aim to provide a fix or mitigation within 7 days for critical issues.

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest release | Yes |
| Older releases | Best effort |

## Security Considerations

### Sparkplug B Has No Security of Its Own

Sparkplug B runs over MQTT and relies on the broker for authentication, authorization and transport encryption. `sparkplugbctl` can publish birth, death, data and command messages, so anyone running it with broker access can affect the infrastructure on that broker.

- **Use `sparkplugbctl` only on brokers and networks you own or are authorized to access**
- Use a `tls://` or `ssl://` broker URL where the broker supports it (`mqtts://`, `wss://`, `mqtt+ssl://` and `tcps://` connect with TLS too). `--tls-ca` sets the CA that verifies the broker, with or without a client certificate; without it the system roots are used. A client certificate needs both `--tls-cert` and `--tls-key`: one of them alone has no effect
- The `--tls-*` files are used only with one of those schemes. With a `tcp://`, `mqtt://`, `ws://` or `unix://` broker URL, or one without a scheme, the connection, and the username and password with it, is not encrypted, whatever `--tls-*` says: each command that connects prints a warning on stderr naming the flag and the scheme, and connects without TLS

### Credential Handling

- `listen`, `publish`, `edge`, `host` and `app` log in with `--username` and a password from `--password-file`, `SPARKPLUGBCTL_PASSWORD_FILE`, `SPARKPLUGBCTL_PASSWORD` or `--password`
- Prefer `--password-file` (or `SPARKPLUGBCTL_PASSWORD_FILE`): a file only its owner can read (`chmod 600`), such as one a secret manager mounts. The whole file is the password, but for one trailing `\n` or `\r\n`; it is read only by the commands that connect, and holds at most the 65535 bytes MQTT allows
- `SPARKPLUGBCTL_PASSWORD` is inherited by the processes the shell starts afterwards. `--password` is deprecated: it can appear in process listings and shell history
- MQTT carries a password only together with a username: a password without `--username` is not sent, and a warning on stderr says so
- A username and password in the user information of `--broker` (`tcp://user:pass@host:1883`) are used by the MQTT client to log in. The commands print the broker URL with the password replaced by `xxxxx`; still, the URL is on the command line, so prefer `--username` with `--password-file`
- Neither the password nor the password file's content is printed, on stdout or stderr
- Client private keys (`--tls-key`) should have restrictive file permissions (`chmod 600`)

### Data From the Broker Is Untrusted Input

Payloads received from a broker come from other clients. Treat what `sparkplugbctl` prints as untrusted: validate values before acting on them, and escape strings before putting them in HTML, SQL or shell commands.

- **Payload size.** `listen` and `decode` decode a payload of at most 16 MiB (16777216 bytes), far above what a Sparkplug payload carries in practice; `--max-payload` sets another bound. `listen` reports a larger message on stderr, as a payload that does not decode, and goes on; `decode` refuses a larger input with exit 3 after reading one byte more than the bound, so a device or a pipe that does not end is not read to its end. The MQTT client receives a whole message before `listen` sees it: bound the message size at the broker as well.
- **Memory.** Decoding a payload takes memory in proportion to its size: up to about 90 times it on a 64-bit platform for a payload of many empty metrics, and less for metrics that carry values. Where memory is short, lower `--max-payload`. Printing adds little to that: the metrics of a message are formatted and written one at a time, the text form in blocks of at most 4096 lines or 256 KiB, so the output of a message is not held in memory; the largest a single metric's output can be is set by the go-sparkplugb limits of 1 << 20 array elements and 16 MiB strings.
- **Terminal.** In text output, group, edge node, device and metric names, STATE host IDs and string values are printed with their control characters (C0, DEL and C1), bidirectional formatting characters, line and paragraph separators and bytes that are not UTF-8 replaced by Go escapes (`\x1b`, `\u202e`, `\xff`); other characters, and names without such characters, are printed as they are. `--json` output carries the strings as they arrived, escaped only as JSON requires: escape them before writing them to a terminal.

### Static Analysis

Gosec and Opengrep do not run in public CI (otfabric/engineering ADR-0011, rule GO-014): a maintainer scans sparkplugbctl on their own machine with `cd ../engineering && make sast REPO=sparkplugbctl` in a checkout of [otfabric/engineering](https://github.com/otfabric/engineering) beside this one before each release. Gosec findings are fixed in the code: there is no `nolint` or `#nosec` comment and no lint exclusion. A finding that is accepted instead is recorded in a private triage repository, never in this one: the files `decode --file` and `--password-file` read are the ones the operator names, and the simulated values of `edge` draw from `math/rand/v2`. `encode --file` creates its output file for the owner only (`0600`); TLS always verifies the broker.
