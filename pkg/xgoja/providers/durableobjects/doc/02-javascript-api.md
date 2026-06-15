---
Title: "Durable Objects JavaScript API"
Slug: "go-go-objects-js-api"
Short: "Write JavaScript object classes, RPC methods, fetch handlers, storage calls, and alarms."
Topics:
- durable-objects
- javascript
- rpc
- fetch
- storage
- alarms
- promises
Commands:
- go-go-objects
Flags:
- bundle
- manifest
- cpu-timeout
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

A Durable Objects bundle is a CommonJS JavaScript file that exports object classes through `exports.objects`. Each class instance represents one object identity and receives a `state` object and an `env` object in its constructor.

The API intentionally covers the core local runtime model first: synchronous SQLite-backed storage, RPC methods, fetch handlers, alarms, and Promise-aware dispatch. It does not yet implement Cloudflare input gates, output gates, or async storage APIs.

## Bundle shape

Export classes from `exports.objects`.

```javascript
class Counter {
  constructor(state, env) {
    this.state = state;
    this.env = env;
  }

  async increment(by = 1) {
    const current = this.state.storage.get("count") || 0;
    const next = current + by;
    this.state.storage.put("count", next);
    return next;
  }
}

exports.objects = { Counter };
```

The default namespace for `Counter` is `COUNTER`. The runtime derives namespaces by converting exported class names from CamelCase to upper snake case.

## RPC methods

Any method on the object instance can be called through the RPC gateway.

```bash
curl -X POST http://127.0.0.1:8787/rpc/COUNTER/global/increment \
  -H 'content-type: application/json' \
  -d '[3]'
```

RPC request bodies are JSON arrays. The return value is encoded as JSON in a response envelope.

```json
{"ok":true,"result":3}
```

RPC methods may return plain values or Promises. The runtime waits until a returned Promise is fulfilled, rejected, or the dispatch timeout expires.

## Fetch handlers

Define `fetch(req)` to handle HTTP-style object requests.

```javascript
class Counter {
  fetch(req) {
    if (req.path === "/count") {
      return {
        status: 200,
        headers: { "content-type": "text/plain" },
        body: String(this.state.storage.get("count") || 0),
      };
    }
    return { status: 404, body: "not found" };
  }
}
```

The gateway path is:

```text
/fetch/:namespace/:objectName/*path
```

The `req` object contains:

| Field | Meaning |
| --- | --- |
| `method` | HTTP method. |
| `url` | Full request URL string. |
| `path` | Object-local path after namespace and object name. |
| `query` | Raw query string. |
| `headers` | Header map. |
| `body` | Text body when available. |
| `rawBody` | Raw request bytes bridged as a Go value. |

Fetch handlers may also be `async` or return a Promise.

## Storage API

`state.storage` is a synchronous key-value API backed by SQLite.

| Method | Behavior |
| --- | --- |
| `get(key)` | Return the stored value or `undefined`. |
| `put(key, value)` | Store a JSON-serializable value. |
| `delete(key)` | Delete one key. |
| `list({ prefix })` | Return keys and values, optionally filtered by literal prefix. |
| `transaction(fn)` | Run a synchronous callback in a SQLite transaction. |
| `setAlarm(timestampMs)` | Schedule this object's alarm. |
| `getAlarm()` | Return the scheduled alarm timestamp or `undefined`. |
| `deleteAlarm()` | Clear the scheduled alarm. |

`transaction(fn)` must be synchronous. If the callback returns a pending Promise, the runtime rejects it because holding a SQLite transaction open across an await would be unsafe.

## Alarms

Define `alarm()` to handle scheduled alarms.

```javascript
class Counter {
  async alarm() {
    const current = this.state.storage.get("alarmCount") || 0;
    this.state.storage.put("alarmCount", current + 1);
  }
}
```

Alarm handlers may return Promises. The Go alarm scheduler dispatches due alarms through the same actor serialization path as RPC and fetch.

## Async semantics

Returned Promises are awaited for RPC, fetch, and alarm handlers. Dispatches to the same object are serialized for the full event lifecycle: method invocation, Promise settlement, and result conversion.

The dispatch timeout configured by `--cpu-timeout` is currently a total JavaScript CPU and Promise settlement budget. If a Promise never settles, the request fails with a timeout.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| Async method returns `{}` or an unresolved object | The runtime is too old to await handler Promises. | Use a version with Promise-aware dispatch. |
| `transaction callback must be synchronous` | A storage transaction returned a Promise. | Move async work outside the transaction. |
| Prefix list matches too many keys | Older versions treated `%` and `_` as SQL wildcards. | Use the current version; prefixes are escaped literally. |
| Concurrent async increments lose updates | Object dispatches are not serialized in the runtime version. | Use the current version with per-actor async dispatch serialization. |

## See also

- `go-go-objects-overview`
- `go-go-objects-xgoja-provider`
