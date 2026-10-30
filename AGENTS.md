# AGENTS.md — sparkplugbctl

## What this is

Command-line tool for Sparkplug B 3.0.0 over MQTT built on [go-sparkplugb](https://github.com/otfabric/go-sparkplugb): listen, publish, simulate edge nodes and host applications, encode and decode payloads. Pure Go. Profile: **cli**.

## Organisation standards

This repository follows **otfabric/engineering policy 1.0**, profile **cli** (`.otfabric/policy.yaml`, mode `baseline`). Where this file and the organisation standards disagree, this file wins here — say so in your report.

## Where the truth lives

| Question | Source |
|---|---|
| Exit codes (script-facing API) | docs/exitcodes.md |
| Commands, flags, environment | README.md |
| Security guidance | SECURITY.md |
| Sparkplug behaviour | go-sparkplugb |
| Release history | RELEASE.md (`## vX.Y.Z` … `---`) |

## Hard rules

1. Exit codes and `--json` output are a public contract: changing them is a breaking change with a RELEASE.md note.
2. Never lower `COVER_MIN`, add lint suppressions, or skip or delete tests to make CI pass.
3. go-sparkplugb is pinned to a released tag; no `replace` directives.

## Commands

| When you change… | Run |
|---|---|
| anything | `make check` |
| timing-sensitive tests | `make stress` |

## Definition of Done

`make check` green; tests for new commands; README.md updated for new flags; RELEASE.md section for user-visible changes.

## Git and releases

Branches and PRs. Do not push to `main`, tag or run the Release workflow unless the maintainer asks in the session.
