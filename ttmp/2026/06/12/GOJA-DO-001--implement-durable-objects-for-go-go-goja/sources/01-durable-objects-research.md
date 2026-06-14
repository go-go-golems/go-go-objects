---
Title: Durable Objects Research for go-go-goja
Ticket: GOJA-DO-001
Status: active
Topics:
    - goja
    - durable-objects
    - actor-runtime
DocType: reference
Intent: source-material
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../../../tmp/durable-objects.md
      Note: Original research document imported into ticket sources
ExternalSources: []
Summary: Research document on implementing Cloudflare-style Durable Objects as a pure goja actor runtime.
LastUpdated: 2026-06-12T00:00:00-04:00
WhatFor: Source material for the durable objects design analysis.
WhenToUse: Read this to understand the research basis for the go-go-goja durable objects implementation.
---


A good version is **not “Cloudflare Workers reimplemented in Go.”** It is an **addressable actor runtime** where each object ID maps to one logical actor, that actor owns its in-memory JS state, and it has private durable storage.

Cloudflare’s Durable Objects model is: globally addressable object IDs, colocated durable storage, in-memory state, RPC/fetch dispatch, alarms, and WebSocket coordination. Their docs also emphasize that objects are single-threaded/cooperatively multitasked, with input/output gates to avoid storage races. ([Cloudflare Docs][1]) ([Cloudflare Docs][2]) ([Cloudflare Docs][3])

## Core shape

```text
                 ┌────────────────────────────┐
HTTP / RPC / WS ─▶ Gateway / Router            │
                 └────────────┬───────────────┘
                              │ object namespace + object id
                              ▼
                 ┌────────────────────────────┐
                 │ Object Directory / Manager │
                 │ id -> active actor         │
                 └────────────┬───────────────┘
                              │
                              ▼
                 ┌────────────────────────────┐
                 │ Durable Object Actor       │
                 │ - one goroutine            │
                 │ - one goja.Runtime         │
                 │ - mailbox queue            │
                 │ - in-memory JS instance    │
                 │ - private storage handle   │
                 └────────────┬───────────────┘
                              │
                              ▼
                 ┌────────────────────────────┐
                 │ SQLite / KV / Alarm Store  │
                 └────────────────────────────┘
```

The important implementation decision: **one active actor per object ID**. In Go, that means one mailbox and one owner goroutine per live object. That fits goja because a `goja.Runtime` is not goroutine-safe; one runtime can only be used by one goroutine at a time, and JS object values cannot be passed between runtimes. ([GitHub][4])

## MVP

Build this first:

1. **Single Go binary**
   No cluster. No global placement. No replication. One process, one machine.

2. **One JS bundle**
   Load a bundled JS file at startup. Use esbuild/Babel to target a conservative JS subset. Goja has full ECMAScript 5.1 support and partial newer ECMAScript support, so do not start by promising Cloudflare Worker compatibility. ([GitHub][4])

3. **Object namespaces**
   A manifest maps namespace names to JS classes:

   ```json
   {
     "objects": {
       "COUNTER": "Counter",
       "CHAT_ROOM": "ChatRoom"
     }
   }
   ```

4. **Addressing**
   Support:

   ```text
   POST /rpc/:namespace/:name/:method
   ANY  /fetch/:namespace/:name/*
   ```

   Example:

   ```text
   POST /rpc/COUNTER/global/increment
   ```

5. **Actor manager**
   Lazily start object actors on first request. Keep them in memory while active. Evict after idle timeout.

6. **Per-object storage**
   Start with SQLite. Either:

   * one SQLite file per object, simplest isolation;
   * or one SQLite file per shard with `object_id` prefixes, simpler file management.

   For MVP, I would choose **one SQLite DB per object**. It maps cleanly to Durable Objects’ private storage model. Cloudflare’s current storage API is SQLite-backed for new Durable Objects and includes SQL, KV, PITR, and alarms, with each object’s storage private to that instance. ([Cloudflare Docs][5])

7. **Synchronous KV API**
   Start with:

   ```js
   state.storage.get(key)
   state.storage.put(key, value)
   state.storage.delete(key)
   state.storage.list(prefix)
   ```

   Do not begin with full async `await` semantics. Goja does not provide browser/Node timers by itself; the host must implement the event loop and functions like `setTimeout`. ([GitHub][4])

8. **RPC method dispatch**
   Cloudflare now prefers RPC methods for new projects, where public Durable Object methods are called through stubs and use serializable arguments/results. Your MVP can copy that shape with JSON-only serialization. ([Cloudflare Docs][6])

9. **Alarms**
   Add one alarm per object:

   ```js
   state.storage.setAlarm(Date.now() + 60_000)
   ```

   A Go scheduler scans alarm due times and enqueues an `alarm` message to the object.

10. **No WebSocket hibernation yet**
    Basic WebSockets can come after HTTP/RPC. Hibernation is valuable but not MVP; Cloudflare’s hibernation model keeps clients connected while the object sleeps and wakes it on messages, which is materially harder. ([Cloudflare Docs][7])

## JavaScript authoring model

Do not start with exact Cloudflare syntax. Use a smaller API:

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

  alarm() {
    // scheduled work
  }
}

exports.objects = { Counter };
```

Then later evolve toward Cloudflare-like APIs:

```js
export class Counter extends DurableObject {
  async increment() {}
  async fetch(request) {}
}
```

But that requires a better module loader, Promise/event-loop semantics, and a more complete Web API surface.

## Go implementation architecture

### 1. `Gateway`

Responsible for HTTP parsing and turning requests into envelopes.

```go
type Envelope struct {
    Kind       string // "fetch", "rpc", "alarm", "ws-message"
    Namespace  string
    Name       string
    Method     string
    ArgsJSON   []byte
    Request    *HTTPRequest
    Reply      chan Result
    Deadline   time.Time
}
```

### 2. `ObjectID`

Stable identity.

```go
type ObjectID struct {
    Namespace string
    Name      string
    Hash      string
}
```

Use `sha256(namespace + "\x00" + name)` or similar. Names are user-facing; hashes are storage paths and routing keys.

### 3. `ActorManager`

Owns the active actor map.

```go
type ActorManager struct {
    mu     sync.Mutex
    actors map[ObjectID]*Actor
    bundle *Bundle
}

func (m *ActorManager) Dispatch(ctx context.Context, env Envelope) Result {
    actor := m.getOrStart(ObjectID{
        Namespace: env.Namespace,
        Name:      env.Name,
    })
    return actor.Call(ctx, env)
}
```

### 4. `Actor`

The actor owns exactly one goroutine and one `goja.Runtime`.

```go
type Actor struct {
    id       ObjectID
    inbox    chan Envelope
    vm       *goja.Runtime
    instance *goja.Object
    storage  *Storage
    idleAt   time.Time
}
```

Actor loop:

```go
func (a *Actor) loop() {
    a.initRuntime()

    for env := range a.inbox {
        result := a.handle(env)
        env.Reply <- result
    }
}
```

Inside `handle`, call the JS method:

```go
func (a *Actor) callMethod(method string, args []any) (any, error) {
    fnVal := a.instance.Get(method)
    fn, ok := goja.AssertFunction(fnVal)
    if !ok {
        return nil, fmt.Errorf("method not found: %s", method)
    }

    jsArgs := make([]goja.Value, len(args))
    for i, arg := range args {
        jsArgs[i] = a.vm.ToValue(arg)
    }

    val, err := fn(a.instance, jsArgs...)
    if err != nil {
        return nil, err
    }

    return val.Export(), nil
}
```

### 5. `RuntimeHost`

Installs host APIs into goja:

```go
func installHostAPI(vm *goja.Runtime, state *State, env *Env) {
    vm.Set("console", Console{})
    vm.Set("__state", state)
    vm.Set("__env", env)
}
```

Expose only narrow APIs. Do not expose raw Go filesystem, networking, process, or reflection objects.

### 6. `Storage`

Start with a small KV layer:

```sql
CREATE TABLE IF NOT EXISTS kv (
  key TEXT PRIMARY KEY,
  value BLOB NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS meta (
  key TEXT PRIMARY KEY,
  value BLOB NOT NULL
);
```

Serialization options:

* MVP: JSON.
* Better: CBOR or MessagePack.
* Later: structured clone-like format.

For SQL support, let each object own its own SQLite database and expose:

```js
state.storage.sql.exec("SELECT ...", args)
```

That is powerful but dangerous. Add query timeouts and statement limits.

## Correctness model

Start stricter than Cloudflare.

Cloudflare allows JavaScript async interleaving but uses input/output gates around storage to prevent common data races and to hold outgoing responses until storage writes are safe. ([Cloudflare Docs][3])

For your MVP:

```text
one object = one mailbox = one request runs to completion
```

That means:

* no interleaving inside an object;
* no concurrent access to the goja runtime;
* no storage race inside one object;
* simpler mental model.

This is slower than Cloudflare for long external I/O, but much easier to make correct.

Later, add Cloudflare-like gates:

```text
input gate:
  blocks new messages during protected storage operations

output gate:
  delays responses until storage writes commit
```

## Resource control

You need this early.

Goja supports interruption of running JavaScript via `Runtime.Interrupt`, but the docs note that it only works while executing JS code and does not interrupt native Go functions. ([Go Packages][8]) It also supports `SetMaxCallStackSize`, useful against runaway recursion. ([Go Packages][8])

Use:

```go
timer := time.AfterFunc(cpuBudget, func() {
    vm.Interrupt("cpu time exceeded")
})
defer timer.Stop()
```

Also add:

```go
vm.SetMaxCallStackSize(10_000)
```

For untrusted tenants, do not rely on goja as a hard sandbox. Use process/container isolation, memory limits, syscall restrictions, and killable worker processes. In-process goja is acceptable for trusted or semi-trusted scripts with narrow host APIs.

## MVP API surface

### Object stub

Inside JS:

```js
const stub = env.COUNTER.getByName("global");
const value = stub.rpc("increment", [1]);
```

MVP can make this synchronous. Later make it Promise-based.

### Storage

```js
state.storage.get(key)
state.storage.put(key, value)
state.storage.delete(key)
state.storage.list(prefix)
state.storage.transaction(fn)
```

### Alarms

```js
state.storage.setAlarm(timestampMs)
state.storage.getAlarm()
state.storage.deleteAlarm()
```

### Fetch shape

Avoid implementing full WHATWG `Request`/`Response` initially. Use plain objects:

```js
{
  method: "GET",
  path: "/foo",
  headers: {},
  body: "..."
}
```

Return:

```js
{
  status: 200,
  headers: {},
  body: "..."
}
```

Later, implement standards-compatible `Request`, `Response`, `Headers`, `URL`, `crypto`, and `fetch`.

## Cluster architecture, after MVP

Single-node is easy. Distributed Durable Objects are hard because “only one active instance per ID” must remain true during crashes and partitions.

A practical v1 cluster:

```text
              ┌──────────────┐
              │ Frontends    │
              └──────┬───────┘
                     │
          consistent hash / directory lookup
                     │
                     ▼
              ┌──────────────┐
              │ Object Hosts │
              └──────┬───────┘
                     │
          lease + fencing token
                     │
                     ▼
              ┌──────────────┐
              │ Storage      │
              └──────────────┘
```

Use:

* **consistent hashing** for default placement;
* **directory service** for overrides and active owners;
* **lease table** with `object_id`, `owner_node`, `epoch`, `expires_at`;
* **fencing token** on every storage write;
* **request forwarding** when a frontend contacts the wrong node.

Example lease table:

```sql
CREATE TABLE object_leases (
  object_id TEXT PRIMARY KEY,
  owner_node TEXT NOT NULL,
  epoch INTEGER NOT NULL,
  expires_at INTEGER NOT NULL
);
```

On failover, the new owner increments `epoch`. Storage writes from old epochs are rejected. This prevents split-brain writes after network delays.

For storage:

* easiest cluster: shared Postgres-compatible storage, one logical DB;
* better locality: shard SQLite/Pebble per node and replicate with Raft;
* ambitious: object-level migration with snapshot + WAL replay.

## What not to build in the MVP

Do not start with:

* Cloudflare-compatible Workers runtime;
* full ES modules;
* full async/await/event-loop semantics;
* WebSocket hibernation;
* global edge placement;
* object migration;
* multi-region replication;
* exact structured clone;
* arbitrary npm compatibility.

Those are all second-system traps.

## Suggested build order

### Phase 1: local actor runtime

* Go HTTP server.
* Load JS bundle.
* Start object actor by namespace/name.
* RPC dispatch.
* Sync KV storage.
* Idle eviction.
* CPU timeout.
* Counter demo.

### Phase 2: object lifecycle

* Constructor state restore.
* Alarms.
* Storage transactions.
* Output gate behavior.
* Metrics and logs.
* Crash/restart tests.

### Phase 3: HTTP/fetch compatibility

* Plain request/response first.
* Then `Request`, `Response`, `Headers`, `URL`.
* Add outbound `fetch` only after event loop design is stable.

### Phase 4: WebSockets

* Object as WebSocket coordinator.
* Connections pinned to actor process.
* Broadcast/chat demo.
* No hibernation yet.

### Phase 5: cluster

* Object directory.
* Leases.
* Fencing tokens.
* Request forwarding.
* Node drain/rebalance.
* Storage replication story.

## Minimal demo target

Build a counter and a chat room.

Counter proves:

* object identity;
* single-writer execution;
* durable storage;
* restart recovery.

Chat room proves:

* many clients per object;
* in-memory state;
* message ordering;
* future WebSocket support.

The best MVP definition:

```text
A Go server that can run many named JavaScript actors, route requests to exactly one live actor per ID, persist each actor’s private state, evict idle actors, and wake them for alarms.
```

That is the durable-object kernel. Everything else is compatibility, distribution, and ergonomics.

[1]: https://developers.cloudflare.com/durable-objects/ "Overview · Cloudflare Durable Objects docs"
[2]: https://developers.cloudflare.com/durable-objects/concepts/what-are-durable-objects/ "What are Durable Objects? · Cloudflare Durable Objects docs"
[3]: https://developers.cloudflare.com/durable-objects/best-practices/rules-of-durable-objects/ "Rules of Durable Objects · Cloudflare Durable Objects docs"
[4]: https://github.com/dop251/goja "GitHub - dop251/goja: ECMAScript/JavaScript engine in pure Go · GitHub"
[5]: https://developers.cloudflare.com/durable-objects/api/sqlite-storage-api/ "SQLite-backed Durable Object Storage · Cloudflare Durable Objects docs"
[6]: https://developers.cloudflare.com/durable-objects/best-practices/create-durable-object-stubs-and-send-requests/ "Invoke methods · Cloudflare Durable Objects docs"
[7]: https://developers.cloudflare.com/durable-objects/best-practices/websockets/ "Use WebSockets · Cloudflare Durable Objects docs"
[8]: https://pkg.go.dev/github.com/dop251/goja "goja package - github.com/dop251/goja - Go Packages"

