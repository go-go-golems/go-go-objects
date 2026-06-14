---
Title: "go-go-objects overview"
Slug: "go-go-objects-overview"
Short: "Run Cloudflare-style Durable Objects on top of goja and xgoja."
Topics:
- durable-objects
- goja
- xgoja
- sqlite
- http
Commands:
- go-go-objects
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: Application
---

`go-go-objects` runs Durable Objects-style JavaScript actors inside Go. Each live object owns one goja runtime instance, a SQLite-backed storage namespace, and an HTTP gateway for RPC and fetch-style requests.

Use the standalone binary for local development and smoke testing. Use the xgoja provider when you want Durable Objects embedded into a generated Go/JavaScript host with other xgoja modules.

## Running the built-in counter

The default command starts an HTTP gateway with a built-in counter object when no bundle is supplied.

```bash
go-go-objects --addr 127.0.0.1:8787 --storage ./var/durable-objects
```

Increment the counter through RPC:

```bash
curl -X POST http://127.0.0.1:8787/rpc/COUNTER/global/increment \
  -H 'content-type: application/json' \
  -d '[1]'
```

Read the counter through fetch routing:

```bash
curl http://127.0.0.1:8787/fetch/COUNTER/global/count
```

## Running your own bundle

Provide a CommonJS bundle that exports object classes through `exports.objects`.

```bash
go-go-objects \
  --addr 127.0.0.1:8787 \
  --storage ./var/durable-objects \
  --bundle ./objects.js
```

If no manifest is supplied, namespaces are derived from `exports.objects` class names. For example, `Counter` becomes `COUNTER` and `ChatRoom` becomes `CHAT_ROOM`.

Use `--manifest` only when you need explicit namespace-to-class mappings.

```yaml
objects:
  COUNTER: Counter
  ROOMS: ChatRoom
```

## Runtime behavior

The runtime uses one actor per object identity. Dispatches to the same object are serialized across JavaScript invocation, Promise settlement, and result conversion. This preserves single-object semantics even when handlers are `async`.

SQLite storage is synchronous from JavaScript. Alarms and idle eviction are implemented by Go background loops that dispatch back into the same actor runtime.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `unknown durable object namespace` | The namespace is not in the manifest and could not be derived from `exports.objects`. | Check your export name or add a manifest entry. |
| Request returns 504 | The JavaScript dispatch exceeded `--cpu-timeout`, including Promise settlement time. | Increase `--cpu-timeout` or fix the handler to settle. |
| Storage appears empty | The object identity or `--storage` path changed. | Reuse the same namespace, object name, and storage root. |
| Manifest without bundle fails | A manifest maps namespaces to classes, but classes are loaded from a bundle. | Supply `--bundle` together with `--manifest`. |

## See also

- `go-go-objects-js-api`
- `go-go-objects-xgoja-provider`
