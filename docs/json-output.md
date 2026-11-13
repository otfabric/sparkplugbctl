# `--json` output

`--json` (or `SPARKPLUGBCTL_FORMAT=json`) changes what `decode`, `listen`, `app` and `version` write to stdout. Every other command ignores it: `host` and `edge` always print text, `encode` writes Protobuf bytes, `publish` prints text. Errors are text on stderr, with the exit codes of [exitcodes.md](exitcodes.md); no error object is written to stdout.

Field names are a contract for scripts (`internal/output/output.go`, `appEvent` in `cmd/app.go`, `Info` in `internal/buildinfo`). A change to one is called out in the release notes.

## General

| Concept | Form |
|---|---|
| Names | snake_case |
| Times | RFC 3339 in UTC at second precision, from the millisecond timestamp in the message (`2026-10-18T09:12:03Z`); a timestamp past 2^63-1 ms, which has no date, is its number of milliseconds as text. `app` events are the exception: local time of receipt, with its offset |
| Framing | `decode` and `listen` write indented JSON, one document per payload or message: read a stream with a JSON decoder, not line by line. `app` writes one compact object per line |

## Metric

The `metrics` array of `decode` and `listen` holds objects of this shape:

| Field | Type | Meaning |
|---|---|---|
| `name` | string | The metric name; `alias/<n>` when the message carries only the alias |
| `alias` | number | The alias; omitted when absent or 0 |
| `type` | string | The Sparkplug datatype name without its prefix: `Double`, `Int32`, `Unknown` … |
| `value` | any | The value: number, boolean or string; a `DateTime` is an RFC 3339 string; `Bytes` are base64. `null` when the metric is null or the value cannot be read as its datatype. What JSON cannot hold is text: NaN and the infinities are `"NaN"`, `"+Inf"` and `"-Inf"`, an array holding one is printed as one string, and so is a `DateTime` beyond year 9999 |
| `timestamp` | string | The metric's timestamp; omitted when absent |
| `quality` | string | The `Quality` property: `GOOD`, `BAD`, `STALE`, or the number as a string when it is another value; omitted when absent |

## `decode`

One document:

| Field | Type | Meaning |
|---|---|---|
| `seq` | number | Sequence number; 0 when absent |
| `timestamp` | string | Payload timestamp; omitted when absent |
| `metrics` | array | Metrics; `null` when the payload has none |

```json
{
  "seq": 1,
  "metrics": [
    {
      "name": "temperature",
      "type": "Double",
      "value": 22.5
    }
  ]
}
```

## `listen`

One document per message. Protobuf messages:

| Field | Type | Meaning |
|---|---|---|
| `topic` | string | The MQTT topic |
| `type` | string | `NBIRTH`, `NDATA`, `DDATA` … |
| `group_id` | string | Omitted when empty |
| `node_id` | string | Omitted when empty |
| `device_id` | string | Omitted when empty |
| `seq` | number | Sequence number; 0 when absent |
| `timestamp` | string | Payload timestamp; `""` when absent |
| `metrics` | array | Metrics; `null` when the payload has none |

STATE messages, which are JSON and not Protobuf on the wire:

| Field | Type | Meaning |
|---|---|---|
| `topic` | string | The MQTT topic |
| `type` | string | `STATE` |
| `host_id` | string | The primary host id |
| `online` | boolean | Its state |
| `timestamp` | string | From the message; `""` when absent or 0 |

## `app`

One line per lifecycle or data event:

| Field | Type | Meaning |
|---|---|---|
| `timestamp` | string | When sparkplugbctl received it |
| `type` | string | `NBIRTH`, `DBIRTH`, `NDATA`, `DDATA`, `NDEATH`, `DDEATH` |
| `group_id` | string | |
| `node_id` | string | |
| `device_id` | string | Omitted for node events |
| `metrics` | number | How many metrics the message carried, not the metrics; omitted when 0 |

## `version`

One compact object on one line, the keys always present and in this order:

| Field | Type | Meaning |
|---|---|---|
| `name` | string | `sparkplugbctl` |
| `version` | string | The version without a leading `v`; `dev` for a development build |
| `tag` | string | The git tag of a release build (`v0.4.0`); `""` when there is none |
| `commit` | string | The full commit SHA, with `-dirty` for a build from modified sources; `unknown` when not known |
| `built` | string | The RFC 3339 UTC build time; `unknown` when not known |
| `libraries` | array | The OT Fabric libraries (`github.com/otfabric/go-*`) the binary is built on, sorted by name, each `{"name":"go-sparkplugb","version":"0.3.3"}` with the version without its `v`; `[]` when none |
| `go` | string | The Go version the binary was built with |
| `platform` | string | `<goos>/<goarch>` |

```json
{"name":"sparkplugbctl","version":"0.4.0","tag":"v0.4.0","commit":"1802d0f9afe8642d1e278120b76fcdfac967cb9e","built":"2026-11-05T15:41:49Z","libraries":[{"name":"go-sparkplugb","version":"0.3.3"}],"go":"go1.27.3","platform":"darwin/arm64"}
```
