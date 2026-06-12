# go-go-objects

`go-go-objects` is an experimental Durable Objects runtime built on top of [`goja`](https://github.com/dop251/goja) and the `go-go-goja` runtime owner/engine packages.

The runtime is intentionally small: each object identity maps to one live JavaScript actor, that actor owns one `goja.Runtime`, and each actor has private SQLite-backed durable storage.

## Current capabilities

- Stable object identity: namespace + name + hash
- Manifest mapping namespaces to JavaScript classes
- CommonJS bundle loading with `exports.objects = { Counter }`
- Lazy actor startup through `Manager.Dispatch`
- One owned `goja` runtime per live actor
- Synchronous `state.storage` API backed by SQLite
- JSON-only RPC dispatch
- Plain-object fetch dispatch
- `/rpc/:namespace/:name/:method` and `/fetch/:namespace/:name/*` HTTP gateway
- Persistent alarms with a central SQLite due-alarm index
- Explicit alarm scheduler and idle evictor wrappers
- Idle actor eviction with durable state recovery

## Quick demo

Run the built-in counter demo server:

```bash
go run ./cmd/go-go-objects --addr 127.0.0.1:8787 --storage ./var/durable-objects
```

Increment a durable counter:

```bash
curl -X POST http://127.0.0.1:8787/rpc/COUNTER/global/increment \
  -H 'content-type: application/json' \
  -d '[1]'
```

Read through fetch dispatch:

```bash
curl http://127.0.0.1:8787/fetch/COUNTER/global/count
```

Stop and restart the server, then increment again. The count is recovered from SQLite.

To run your own bundle:

```bash
go run ./cmd/go-go-objects \
  --addr 127.0.0.1:8787 \
  --storage ./var/durable-objects \
  --bundle ./objects.js
```

Namespaces are derived from `exports.objects` keys with Cloudflare-style `CamelCase` to `CAMEL_CASE` conversion. For example, `exports.objects = { ChatRoom }` creates namespace `CHAT_ROOM`; `exports.objects = { Counter }` creates namespace `COUNTER`.

You can still provide an explicit JSON/YAML manifest when you need custom aliases:

```bash
go run ./cmd/go-go-objects \
  --addr 127.0.0.1:8787 \
  --storage ./var/durable-objects \
  --bundle ./objects.js \
  --manifest ./durableobjects.yaml
```

```yaml
objects:
  COUNTER: Counter
  CHAT_ROOM: ChatRoom
```

## JavaScript authoring model

The MVP authoring model is CommonJS:

```js
class Counter {
  constructor(state, env) {
    this.state = state;
    this.env = env;
  }

  increment(by) {
    const current = this.state.storage.get("count") || 0;
    const next = current + (by || 1);
    this.state.storage.put("count", next);
    return next;
  }

  fetch(req) {
    if (req.path === "/count") {
      return { status: 200, body: String(this.state.storage.get("count") || 0) };
    }
    return { status: 404, body: "not found" };
  }
}

exports.objects = { Counter };
```

## Development

```bash
go test ./... -count=1
```

The design and implementation diary live in the docmgr ticket:

```text
ttmp/2026/06/12/GOJA-DO-001--implement-durable-objects-for-go-go-goja
```
