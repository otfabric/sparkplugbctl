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
- Use a `tls://` or `ssl://` broker URL with `--tls-cert` and `--tls-key` where the broker supports it. The TLS settings are applied only when both `--tls-cert` and `--tls-key` are given; `--tls-ca` alone has no effect

### Credential Handling

- Passwords given with `--password` can appear in process listings and shell history; prefer `SPARKPLUGBCTL_PASSWORD`
- Client private keys (`--tls-key`) should have restrictive file permissions (`chmod 600`)

### Data From the Broker Is Untrusted Input

Payloads received from a broker come from other clients. Treat what `sparkplugbctl` prints as untrusted: validate values before acting on them, and escape strings before putting them in HTML, SQL or shell commands.
