# Process exit codes

sparkplugbctl's exit codes are a script-facing contract. Changes are noted in RELEASE.md.

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Any failure: invalid flags, broker connection, encoding or decoding, I/O |

The error is printed on stderr (`main.go`, `cmd/root.go`).
