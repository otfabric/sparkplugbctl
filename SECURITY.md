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
- Use a `tls://` or `ssl://` broker URL where the broker supports it. `--tls-ca` sets the CA that verifies the broker, with or without a client certificate; without it the system roots are used. A client certificate needs both `--tls-cert` and `--tls-key`: one of them alone has no effect

### Credential Handling

- Passwords given with `--password` can appear in process listings and shell history; prefer `SPARKPLUGBCTL_PASSWORD`
- Client private keys (`--tls-key`) should have restrictive file permissions (`chmod 600`)

### Data From the Broker Is Untrusted Input

Payloads received from a broker come from other clients. Treat what `sparkplugbctl` prints as untrusted: validate values before acting on them, and escape strings before putting them in HTML, SQL or shell commands.

### Static Analysis

Gosec and Opengrep do not run in public CI (otfabric/engineering ADR-0011, rule GO-014): a maintainer scans sparkplugbctl on their own machine with `cd ../engineering && make sast REPO=sparkplugbctl` in a checkout of [otfabric/engineering](https://github.com/otfabric/engineering) beside this one before each release. Gosec findings are fixed in the code: there is no `nolint` or `#nosec` comment and no lint exclusion. A finding that is accepted instead is recorded in a private triage repository, never in this one: the file `decode --file` reads is the one the operator names, and the simulated values of `edge` draw from `math/rand/v2`. `encode --file` creates its output file for the owner only (`0600`); TLS always verifies the broker.
