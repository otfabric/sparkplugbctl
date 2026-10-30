# Process exit codes

sparkplugbctl's exit codes are a script-facing contract (`cmd/exit.go`). They follow the organisation table (otfabric/engineering `standards/api-design.md#cli-exit-codes`). The error is printed on stderr.

| Code | Kind | Meaning |
|------|------|---------|
| 0 | ok | Success. `listen`, `host`, `app` and `edge` also exit 0 when stopped with Ctrl-C or SIGTERM |
| 2 | usage | Unknown command or flag, missing required flag, an invalid `SPARKPLUGBCTL_*` value |
| 3 | invalid_input | An unreadable input or TLS file, a payload that does not decode, a metric that does not parse, an invalid configuration |
| 4 | transport | Connecting, subscribing or publishing failed |
| 5 | timeout | A deadline passed |
| 9 | permission | The operating system refused an operation |
| 10 | internal | The output could not be written, or something unexpected happened |
| 130 | interrupted | Stopped by SIGINT or SIGTERM before a result |

`SPARKPLUGBCTL_FORMAT=json` (or `SPARKPLUGBCTL_OUTPUT_FORMAT`; `_FORMAT` wins) has the effect of `--json`.

## Mapping from versions before v0.2.0

Every failure exited 1. Exit 1 is no longer used.
