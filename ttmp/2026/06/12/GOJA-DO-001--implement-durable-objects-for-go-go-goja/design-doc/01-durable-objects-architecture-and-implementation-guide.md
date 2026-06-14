---
Title: Durable Objects Architecture and Implementation Guide
Ticket: GOJA-DO-001
Status: active
Topics:
    - goja
    - architecture
    - durable-objects
    - actor-runtime
    - storage
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../go-go-goja/modules/database/database.go
      Note: |-
        Existing SQLite module and transaction patterns that inform durable object storage implementation.
        SQLite module and transaction implementation reference
    - Path: ../../../../../../../go-go-goja/modules/express/express.go
      Note: |-
        Native module pattern for registering HTTP-facing JavaScript APIs.
        Runtime-aware native module pattern
    - Path: ../../../../../../../go-go-goja/modules/timer/timer.go
      Note: |-
        Async owner-thread settlement pattern for Promise-based timers.
        Async owner-thread settlement pattern for future alarm/timer work
    - Path: ../../../../../../../go-go-goja/pkg/engine/factory.go
      Note: |-
        RuntimeFactory constructs owned goja runtimes and is the starting point for one-runtime-per-actor execution.
        RuntimeFactory creates per-actor goja runtime candidates
    - Path: ../../../../../../../go-go-goja/pkg/engine/runtime.go
      Note: |-
        Runtime lifecycle, closer stack, VM/event-loop/owner tuple, and clean shutdown behavior.
        Runtime lifecycle and close behavior for actor eviction
    - Path: ../../../../../../../go-go-goja/pkg/gojahttp/host.go
      Note: |-
        Existing HTTP-to-JS dispatch path, useful as a reference but not sufficient as the durable object gateway.
        HTTP-to-JS dispatch pattern for gateway design
    - Path: ../../../../../../../go-go-goja/pkg/gojahttp/mountable.go
      Note: Upstream mountable HTTP handler ABI used by Durable Objects
    - Path: ../../../../../../../go-go-goja/pkg/gojahttp/request_response.go
      Note: |-
        Plain request and response DTO model that should be reused for initial fetch dispatch.
        Plain request/response DTO shape
    - Path: ../../../../../../../go-go-goja/pkg/gojahttp/route_registry.go
      Note: |-
        Existing path-pattern matching for :params and * wildcards.
        Route pattern matching reference
    - Path: ../../../../../../../go-go-goja/pkg/runtimebridge/runtimebridge.go
      Note: |-
        Runtime service registry and context propagation used by native modules and async callbacks.
        Runtime service lookup and current owner context for modules
    - Path: ../../../../../../../go-go-goja/pkg/runtimeowner/runner.go
      Note: |-
        Serializes JS execution through Scheduler.RunOnLoop and provides Call/Post semantics for actor dispatch.
        RuntimeOwner Call/Post serialization for actor dispatch
    - Path: ../../../../../../../go-go-goja/pkg/runtimeowner/types.go
      Note: |-
        RuntimeOwner and Scheduler interfaces define the thread-safety contract for goja runtimes.
        RuntimeOwner and Scheduler contracts
    - Path: ../../../../../../../go-go-goja/pkg/xgoja/app/command_providers.go
      Note: Command provider context now passes HostServices for embedded asset resolution
    - Path: ../../../../../../../go-go-goja/pkg/xgoja/app/host_services.go
      Note: |-
        Host service bag and closer contribution pattern for generated xgoja applications.
        xgoja host service contribution and lookup pattern
    - Path: ../../../../../../../go-go-goja/pkg/xgoja/providers/http/http.go
      Note: |-
        Provider packaging pattern, config sections, runtime entries, and external host service injection.
        xgoja provider packaging and service injection model
    - Path: README.md
      Note: |-
        Updated project usage documentation
        Documented custom bundle and manifest usage
        Documents xgoja path and embedded asset configuration
    - Path: cmd/go-go-objects/main.go
      Note: |-
        Built-in counter demo server
        CLI now supports external bundle and manifest loading plus scheduler interval flags
        CLI manifest is now optional
    - Path: cmd/go-go-objects/main_test.go
      Note: CLI bundle/manifest loading tests
    - Path: docs/release-notes.md
      Note: Release checklist and known limitations
    - Path: examples/counter/verbs/site.js
      Note: JS composition layer mounts durableobjects.gateway() into Express
    - Path: examples/counter/xgoja-buildspec.yaml
      Note: xgoja/v2 generated binary example using HTTP serve plus direct durableobjects serve
    - Path: examples/templates/durableobjects_http_runtime.go.tmpl
      Note: Custom xgoja template example for existing http.Server integration
    - Path: go-go-goja/ttmp/2026/06/12/GOJA-DO-001--implement-durable-objects-for-go-go-goja/sources/01-durable-objects-research.md
      Note: Imported source research driving MVP scope and architecture
    - Path: pkg/durableobjects/actor.go
      Note: |-
        Initial actor implementation with runtime bootstrap
        Actor activity tracking and idle detection
    - Path: pkg/durableobjects/alarms.go
      Note: |-
        Alarm index and due alarm dispatch support
        SQLite alarm index reconciliation
    - Path: pkg/durableobjects/bundle.go
      Note: Bundle-derived manifest support from exports.objects keys
    - Path: pkg/durableobjects/durableobjects_test.go
      Note: |-
        Counter persistence and gateway tests
        Alarm dispatch and idle eviction tests
    - Path: pkg/durableobjects/gateway.go
      Note: Initial HTTP gateway for /rpc and /fetch dispatch
    - Path: pkg/durableobjects/id.go
      Note: Namespace and object-name safety validation
    - Path: pkg/durableobjects/manager.go
      Note: |-
        Initial manager implementation with live actor map
        Manager lifecycle context
        Manager now derives manifest from bundle when manifest is omitted
        Singleflight actor startup
    - Path: pkg/durableobjects/manifest.go
      Note: CamelCase to CAMEL_CASE namespace derivation
    - Path: pkg/durableobjects/modules.go
      Note: Actor-local state
    - Path: pkg/durableobjects/scheduler.go
      Note: Explicit alarm scheduler and idle evictor wrappers
    - Path: pkg/durableobjects/server.go
      Note: Embeddable Durable Objects HTTP server helper
    - Path: pkg/durableobjects/storage_sqlite.go
      Note: |-
        Initial SQLite-backed per-object storage implementation
        SQLite metadata and schema versioning
    - Path: pkg/xgoja/providers/durableobjects/durableobjects.go
      Note: |-
        xgoja provider package with config capability
        xgoja provider manifest-path is now optional
        xgoja provider now supports filesystem and embedded asset bundle configuration
        Unified provider config
        Exports mountable durableobjects.gateway() handler for xgoja/v2 HTTP serve composition
    - Path: pkg/xgoja/providers/durableobjects/durableobjects_test.go
      Note: |-
        xgoja provider registration
        Provider tests cover embedded bundle assets and mixed mode validation
    - Path: pkg/xgoja/providers/durableobjects/serve.go
      Note: xgoja durableobjects serve command provider
ExternalSources:
    - https://developers.cloudflare.com/durable-objects/
    - https://developers.cloudflare.com/durable-objects/concepts/what-are-durable-objects/
    - https://developers.cloudflare.com/durable-objects/best-practices/rules-of-durable-objects/
    - https://developers.cloudflare.com/durable-objects/api/sqlite-storage-api/
    - https://github.com/dop251/goja
Summary: Evidence-backed implementation guide for adding a go-go-goja Durable Objects runtime as an addressable actor system with one owned goja runtime per live object, private SQLite-backed storage, RPC/fetch dispatch, and alarms.
LastUpdated: 2026-06-12T16:35:00-04:00
WhatFor: Use this as the intern-facing design and implementation guide for GOJA-DO-001.
WhenToUse: Read before implementing durable object packages, storage, gateway routing, xgoja provider integration, tests, or follow-on compatibility work.
---

















# Durable Objects Architecture and Implementation Guide

## Executive summary

This ticket should implement a **Durable Objects kernel** for `go-go-goja`: a local, single-process actor runtime where each object identity maps to exactly one live JavaScript object instance at a time, and that instance owns both its in-memory state and a private durable storage handle. The goal is not to re-create Cloudflare Workers. The goal is to build the small, correct core that makes Durable Objects useful: addressable actors, single-writer execution, persistent per-object state, request routing, and alarms.

The existing `go-go-goja` codebase already contains most of the hard runtime primitives. `engine.RuntimeFactory.NewRuntime()` constructs a `goja.Runtime`, a Node-style event loop, a `RuntimeOwner`, a require registry, runtime services, and cleanup hooks (`pkg/engine/factory.go:184-260`). `runtimeowner.RuntimeOwner` serializes calls through a `Scheduler.RunOnLoop` interface (`pkg/runtimeowner/types.go:9-27`) and implements request/response dispatch with `Call()` (`pkg/runtimeowner/runner.go:90-134`). `gojahttp.Host` already shows how to route Go HTTP requests into JavaScript callbacks through `RuntimeOwner.Call()` (`pkg/gojahttp/host.go:94-163`). The database module already exposes SQLite query, exec, and transaction primitives to JavaScript (`modules/database/database.go:220-237`, `modules/database/database.go:278-373`).

The new work should therefore be a focused package set, not a rewrite of the runtime:

1. `pkg/durableobjects` for core object identity, manager, actor, storage, gateway, and alarm scheduler.
2. `modules/durableobjects` or an internal native-module registrar for JavaScript-visible `state`, `env`, and object stubs.
3. `pkg/xgoja/providers/durableobjects` so generated xgoja binaries can opt into the runtime the same way the HTTP provider opts into `express`.
4. Minimal examples and tests: a counter object and a chat-room-shaped object without WebSocket hibernation.

The first implementation must stay deliberately small: CommonJS bundle loading, synchronous JSON-only RPC, plain-object fetch request/response shapes, one SQLite database per object, one request runs to completion per object, one alarm per object, and no cluster behavior. This is the right MVP because it gives a simple correctness model and fits goja's rule that a `goja.Runtime` is not safe to use concurrently.

## Problem statement and scope

Durable Objects solve a common problem in JavaScript-hosted applications: many clients need to address the same logical stateful object, and that object must process messages in a predictable order while keeping durable private state. Examples are counters, document sessions, chat rooms, workflow coordinators, per-user agents, and locks.

In `go-go-goja`, the missing abstraction is not JavaScript execution. The repository already has robust JavaScript execution infrastructure. The missing abstraction is **identity-bound runtime ownership**:

- A caller says: "send this RPC request to namespace `COUNTER`, object name `global`, method `increment`."
- The host resolves this to a stable `ObjectID`.
- The manager lazily starts the actor if it is not already live.
- The actor executes the method against the same JavaScript instance and same private storage every time.
- The actor can be evicted after idleness and later reconstructed from storage.

### In scope for the first implementation

The implementation should include:

- A single-process `Manager` with a map from `ObjectID` to live `Actor`.
- A stable identity model: namespace, name, and hash.
- A manifest mapping namespace names to JavaScript class names.
- Loading one CommonJS bundle that exports object classes.
- RPC dispatch over JSON.
- Fetch dispatch over a plain request/response DTO.
- Per-object SQLite-backed storage with synchronous KV helpers.
- One persistent alarm timestamp per object and a Go scheduler that wakes due objects.
- Idle eviction and runtime cleanup.
- CPU/runtime budget support using `goja.Runtime.Interrupt`.
- xgoja provider integration after the core package works.

### Out of scope for the first implementation

The first implementation must not attempt:

- Cloudflare Workers API compatibility.
- Full WHATWG `Request`, `Response`, `Headers`, `URL`, `fetch`, or `crypto.subtle` compatibility.
- Full async/await semantics for storage.
- ES module loading as the primary format.
- WebSocket hibernation.
- Distributed placement, object migration, leases, replication, or multi-node routing.
- Arbitrary npm package compatibility.
- A hard in-process security sandbox for untrusted tenants.

These are follow-on projects. The MVP should produce a reliable local actor runtime first.

## Terms and mental model

An intern should internalize these terms before touching code.

- **Durable object:** A named, addressable JavaScript actor with private durable storage.
- **Namespace:** A logical collection of objects that all instantiate the same JavaScript class. Example: `COUNTER` maps to class `Counter`.
- **Object name:** A user-facing string inside a namespace. Example: `global`, `room-123`, or `user:42`.
- **ObjectID:** The stable Go identity derived from namespace and name, plus a hash used for storage paths.
- **Manager:** The Go object that owns the live actor map, starts actors, routes messages, and evicts idle actors.
- **Actor:** One live object instance. It owns exactly one `engine.Runtime`, one JavaScript instance, one storage handle, and one logical mailbox via `RuntimeOwner.Call()`.
- **Gateway:** The HTTP-facing adapter that converts `/rpc/...` and `/fetch/...` requests into manager dispatch calls.
- **Storage:** A per-object SQLite database with KV, metadata, and alarm tables.
- **Alarm scheduler:** A Go background component that scans storage for due alarms and dispatches `alarm` messages to the manager.

The shape is:

```text
                       Go HTTP server / mux
                              │
                              ▼
                 ┌───────────────────────────┐
                 │ durableobjects.Gateway    │
                 │ /rpc and /fetch routes    │
                 └─────────────┬─────────────┘
                               │ Envelope{ObjectID, Kind, ...}
                               ▼
                 ┌───────────────────────────┐
                 │ durableobjects.Manager    │
                 │ live map: ObjectID→Actor  │
                 │ lazy start + eviction     │
                 └─────────────┬─────────────┘
                               │ Actor.Call(ctx, env)
                               ▼
                 ┌───────────────────────────┐
                 │ durableobjects.Actor      │
                 │ engine.Runtime            │
                 │ JS class instance         │
                 │ private Storage           │
                 └─────────────┬─────────────┘
                               │
                               ▼
                 ┌───────────────────────────┐
                 │ SQLite object database    │
                 │ kv, meta, alarm           │
                 └───────────────────────────┘
```

## Current-state analysis with evidence

### Runtime construction already matches actor needs

`engine.Runtime` is the existing unit of owned JavaScript execution. It contains the VM, require module, event loop, owner, values, lifetime context, and closer list (`pkg/engine/runtime.go:32-47`). Its `Close()` path cancels runtime context, waits for the owner to become idle or interrupts active JavaScript, runs registered closers, deletes runtimebridge services, shuts down the owner, and stops the event loop (`pkg/engine/runtime.go:86-127`). That is exactly the cleanup sequence needed when an actor is evicted.

`RuntimeFactory.NewRuntime()` creates a fresh `goja.Runtime`, a new `eventloop.EventLoop`, starts the loop, creates a `RuntimeOwner`, stores runtimebridge services, registers runtime modules, enables CommonJS `require`, installs console/buffer/url/performance helpers, and then runs runtime initializers (`pkg/engine/factory.go:184-260`). This should be reused rather than bypassed. The durable objects actor should not manually call `goja.New()` unless the existing factory proves insufficient.

### RuntimeOwner is the serialization boundary

The `runtimeowner.Scheduler` interface has one method: `RunOnLoop(fn func(*goja.Runtime)) bool` (`pkg/runtimeowner/types.go:9-12`). `RuntimeOwner.Call()` uses that scheduler to run the callback on the owner loop, propagates context, captures a result or error, and returns to the caller (`pkg/runtimeowner/runner.go:90-134`). `RuntimeOwner.Post()` provides fire-and-forget dispatch for async callbacks (`pkg/runtimeowner/runner.go:136-160` and following lines).

This means the actor does not need to implement a separate goroutine and channel unless the design later needs explicit queue instrumentation. The existing event loop is already the owner queue. The manager can call `actor.runtime.Owner.Call(ctx, "durable-object.rpc", fn)` and the VM work will be serialized.

### gojahttp provides useful request DTOs but is not the durable gateway

`gojahttp.Host` owns a route registry, static mounts, and a `runtimeowner.RuntimeOwner` (`pkg/gojahttp/host.go:26-33`). In `ServeHTTP`, it matches the route, builds a `RequestDTO`, then invokes the registered JavaScript handler through `h.owner.Call()` (`pkg/gojahttp/host.go:116-147`). This is strong precedent for host-to-JS dispatch.

However, durable objects need a gateway that resolves namespace/name/method before entering JavaScript. `gojahttp.Host` routes into one runtime. Durable Objects route into many runtimes, selected by object identity. Reusing the plain DTO from `gojahttp.NewRequestDTO()` is appropriate (`pkg/gojahttp/request_response.go:17-73`), but the gateway itself should be new.

### Existing path matching is enough for MVP routing

`gojahttp.Registry.Match()` already supports exact segments, `:param` segments, and `*` wildcard segments (`pkg/gojahttp/route_registry.go:47-111`). If the Durable Objects gateway is mounted inside a Go mux, it can either use this registry or implement equivalent small parsing. The required MVP routes are simple:

```text
POST /rpc/:namespace/:name/:method
ANY  /fetch/:namespace/:name/*
```

### SQLite and transactions already have JavaScript exposure patterns

The existing database module is more general than durable object storage, but it demonstrates several reusable implementation patterns:

- A module can be preconfigured from Go via `WithPreconfiguredDB()` and `WithCloseFn()` (`modules/database/database.go:80-94`).
- The module exports JavaScript functions using `modules.SetExport()` (`modules/database/database.go:220-237`).
- Context-aware query and exec functions use `runtimebridge.CurrentOwnerContext(vm)` to inherit the current request context (`modules/database/database.go:224-235`).
- Transactions are represented by a Go-backed `TransactionHandle` exposed as a JavaScript object (`modules/database/database.go:360-390`).

The durable storage API should not initially expose the general `database` module directly. It should expose a narrower `state.storage` API backed by SQLite. SQL can be added later as `state.storage.sql.exec(...)` after query limits and authorization are designed.

### The native module pattern is simple and should be followed

A native module implements `Name()`, `Doc()`, and `Loader(*goja.Runtime, *goja.Object)` (`modules/common.go:28-32`). The Express module shows a more advanced runtime-aware pattern: it receives a `gojahttp.Host`, wires the host to the runtime owner, registers a CommonJS module, and exposes functions that register route handlers (`modules/express/express.go:55-77`, `modules/express/express.go:132-146`).

Durable Objects needs the same idea, but actor-specific: each actor runtime must receive a `state` object and an `env` object bound to the current `ObjectID`, storage handle, manager, and namespace manifest.

### The xgoja provider architecture is the right integration layer

The xgoja HTTP provider is the model for packaging this as a generated-runtime capability. It registers package ID `go-go-goja-http`, contributes a module named `express`, provides TypeScript declarations, adds a config section, and stores per-runtime entries by `*goja.Runtime` (`pkg/xgoja/providers/http/http.go:23-79`). It also consumes a typed external host service from the generated app host service bag (`pkg/xgoja/providers/http/http.go:124-168`).

The host services mechanism supports arbitrary typed service values by key (`pkg/xgoja/app/host_services.go:30-45`, `pkg/xgoja/app/host_services.go:164-188`). Durable Objects should use the same pattern so a Go application can inject a storage root, an external gateway host, or custom configuration.

## Gap analysis

The current repository lacks these components:

| Needed component | Existing support | New work required |
| --- | --- | --- |
| Stable object identity | None | `ObjectID`, validation, hash, storage path derivation |
| Object namespace manifest | xgoja config infrastructure exists | Manifest parser and runtime configuration model |
| Actor manager | RuntimeFactory exists | Live actor map, lazy start, lifecycle, idle eviction |
| Actor wrapper | `engine.Runtime` exists | JS bundle loading, instance construction, dispatch, CPU budgets |
| Object storage | SQLite module exists | Narrow KV API, object DB schema, alarm persistence |
| Object stubs | None | JavaScript `env.NAMESPACE.getByName(name).rpc(method,args)` |
| Gateway | `gojahttp.Host` pattern exists | Multi-runtime routing for `/rpc` and `/fetch` |
| Alarm scheduler | `timer` module exists | Persistent alarm scanning and manager dispatch |
| Provider integration | HTTP provider pattern exists | `pkg/xgoja/providers/durableobjects` |
| Tests/examples | Many module/provider tests | Counter, storage, restart, eviction, alarms, gateway |

## Proposed architecture

### Package layout

Use clear package boundaries. The names below intentionally separate core runtime from xgoja provider glue.

```text
pkg/durableobjects/
  id.go                 // ObjectID, namespace/name validation, hash
  manifest.go           // Namespace manifest and JS class lookup
  envelope.go           // Dispatch envelopes and results
  manager.go            // live actor map, lazy start, eviction
  actor.go              // per-object runtime and JS instance
  bundle.go             // CommonJS bundle loading and exports.objects lookup
  storage.go            // Storage interface and SQLite implementation
  storage_sqlite.go     // SQLite schema, KV serialization, alarm metadata
  gateway.go            // net/http Handler for /rpc and /fetch
  alarms.go             // alarm scheduler
  modules.go            // actor-local state/env module registrars
  errors.go             // typed errors and HTTP mapping
  metrics.go            // optional counters/log fields, can start small

pkg/xgoja/providers/durableobjects/
  durableobjects.go     // provider registration
  config.go             // xgoja/glazed config sections
  serve.go              // optional command integration after core works
  typescript.go         // JS API declarations

examples/durableobjects/
  counter.js
  durableobjects.yaml
```

Do not put the core manager into `modules/`. `modules/` is for JavaScript-visible CommonJS modules. The manager is a Go runtime service and belongs under `pkg/durableobjects`.

### Runtime composition

Each live object gets its own `engine.Runtime`. This is non-negotiable for correctness and isolation: goja values cannot be shared between runtimes, and a runtime must not be used concurrently. The actor owns the runtime until eviction.

The key implementation choice is how to inject actor-specific state into a runtime. The existing `RuntimeFactory` is immutable after build. Actor-specific modules need access to the current actor's storage and manager. Therefore the recommended MVP is:

1. `Manager` owns a base `RuntimeFactoryBuilder` configuration or a small `RuntimeBuilder` helper.
2. When starting an actor, create actor-specific module registrars that close over `ObjectID`, `Storage`, and `Manager`.
3. Build a per-actor factory and call `NewRuntime()`.
4. Register closers for storage and actor resources.
5. Load the bundle and instantiate the configured JS class.

This adds a little startup overhead per actor, but it keeps the API simple and avoids modifying the existing engine package before the design has proven itself.

Pseudocode:

```go
func (m *Manager) startActor(ctx context.Context, id ObjectID) (*Actor, error) {
    className, ok := m.manifest.ClassForNamespace(id.Namespace)
    if !ok { return nil, ErrUnknownNamespace }

    storage, err := m.storageFactory.Open(ctx, id)
    if err != nil { return nil, err }

    actor := &Actor{id: id, manager: m, storage: storage, className: className}

    factory, err := engine.NewRuntimeFactoryBuilder(
        engine.WithImplicitDefaultRegistryModules(false),
        engine.WithDataOnlyDefaultRegistryModules(true),
    ).WithModules(
        durableobjects.StateRegistrar(actor),
        durableobjects.EnvRegistrar(m, id),
    ).UseModuleMiddleware(
        engine.MiddlewareOnly("crypto", "events", "path", "time", "timer"),
    ).Build()
    if err != nil { return nil, err }

    rt, err := factory.NewRuntime(engine.WithLifetimeContext(m.ctx))
    if err != nil { _ = storage.Close(); return nil, err }
    actor.runtime = rt
    _ = rt.AddCloser(func(context.Context) error { return storage.Close() })

    if err := actor.bootstrap(ctx, m.bundle, className); err != nil {
        _ = rt.Close(ctx)
        return nil, err
    }
    return actor, nil
}
```

If per-actor factory construction proves too expensive, a later optimization can add a runtime option for module overlays. Do not start with that engine change unless profiling shows it matters.

### Core types

#### ObjectID

```go
type ObjectID struct {
    Namespace string `json:"namespace"`
    Name      string `json:"name"`
    Hash      string `json:"hash"`
}

func NewObjectID(namespace, name string) (ObjectID, error) {
    namespace = strings.TrimSpace(namespace)
    name = strings.TrimSpace(name)
    if namespace == "" { return ObjectID{}, ErrEmptyNamespace }
    if name == "" { return ObjectID{}, ErrEmptyName }
    sum := sha256.Sum256([]byte(namespace + "\x00" + name))
    return ObjectID{Namespace: namespace, Name: name, Hash: hex.EncodeToString(sum[:])}, nil
}
```

Use `Hash` for filesystem paths and metrics labels that need bounded characters. Preserve `Namespace` and `Name` for logs and JS-visible state.

#### Manifest

```go
type Manifest struct {
    Objects map[string]string `json:"objects" yaml:"objects"`
}

func (m Manifest) ClassForNamespace(namespace string) (string, bool) {
    className, ok := m.Objects[namespace]
    return className, ok
}
```

Initial manifest example:

```yaml
objects:
  COUNTER: Counter
  CHAT_ROOM: ChatRoom
```

This should be usable in tests without xgoja. Later, the xgoja provider can map this into generated config sections.

#### Envelope and Result

```go
type Kind string

const (
    KindRPC   Kind = "rpc"
    KindFetch Kind = "fetch"
    KindAlarm Kind = "alarm"
)

type Envelope struct {
    Kind      Kind
    ID        ObjectID
    Method    string
    ArgsJSON  json.RawMessage
    Request   *FetchRequest
    Deadline  time.Time
    RequestID string
}

type Result struct {
    ValueJSON json.RawMessage
    Response  *FetchResponse
    Error     error
}
```

Keep the envelope Go-native. Do not expose `goja.Value` through manager APIs. Values must not cross runtime boundaries.

#### Manager

```go
type Manager struct {
    mu       sync.Mutex
    actors   map[ObjectID]*Actor
    manifest Manifest
    bundle   *Bundle
    storage  StorageFactory
    opts     Options
    alarms   *AlarmScheduler
}

func (m *Manager) Dispatch(ctx context.Context, env Envelope) (Result, error) {
    actor, err := m.getOrStart(ctx, env.ID)
    if err != nil { return Result{}, err }
    return actor.Dispatch(ctx, env)
}
```

`getOrStart` must hold the map lock only while checking/inserting. It must not hold the lock while creating the runtime or running JavaScript. Use a start-in-progress entry or `singleflight` to avoid duplicate starts for the same ID under concurrent first requests.

#### Actor

```go
type Actor struct {
    id        ObjectID
    className string
    runtime   *engine.Runtime
    instance  *goja.Object // only touch on owner thread
    storage   Storage
    manager   *Manager
    lastUsed  atomic.Int64
    closing   atomic.Bool
}

func (a *Actor) Dispatch(ctx context.Context, env Envelope) (Result, error) {
    a.touch()
    return a.withBudget(ctx, func(ctx context.Context) (Result, error) {
        ret, err := a.runtime.Owner.Call(ctx, "durable-object."+string(env.Kind), func(ctx context.Context, vm *goja.Runtime) (any, error) {
            return a.dispatchOnOwner(ctx, vm, env)
        })
        if err != nil { return Result{}, err }
        return ret.(Result), nil
    })
}
```

`instance` is only read or written inside `RuntimeOwner.Call()`. This should be documented in comments to prevent future data races.

### Bundle loading and JavaScript class construction

For MVP, use a CommonJS bundle that sets `exports.objects = { Counter, ChatRoom }`.

Example JavaScript:

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
    this.state.storage.put("alarmCount", (this.state.storage.get("alarmCount") || 0) + 1);
  }
}

exports.objects = { Counter };
```

Bootstrap pseudocode:

```go
func (a *Actor) bootstrap(ctx context.Context, bundle *Bundle, className string) error {
    _, err := a.runtime.Owner.Call(ctx, "durable-object.bootstrap", func(ctx context.Context, vm *goja.Runtime) (any, error) {
        moduleExports, err := bundle.Evaluate(ctx, vm, a.runtime.Require)
        if err != nil { return nil, err }

        objects := moduleExports.ToObject(vm).Get("objects").ToObject(vm)
        ctorVal := objects.Get(className)
        ctor, ok := goja.AssertConstructor(ctorVal)
        if !ok { return nil, fmt.Errorf("object class %q is not a constructor", className) }

        state := newStateObject(vm, a.id, a.storage)
        env := newEnvObject(vm, a.manager, a.id)
        instance, err := ctor(goja.Undefined(), state, env)
        if err != nil { return nil, err }
        a.instance = instance.ToObject(vm)
        return nil, nil
    })
    return err
}
```

If `goja.AssertConstructor` does not match the installed goja API, adapt this to the exact constructor call API used by the current version. The important invariant is: construct once per actor lifetime, keep the instance, dispatch methods on that instance.

### RPC dispatch

RPC accepts JSON arguments and returns a JSON-serializable result. The gateway should accept either a raw JSON array or an object with `args`:

```json
{ "args": [1] }
```

Actor pseudocode:

```go
func (a *Actor) callRPC(vm *goja.Runtime, method string, argsJSON []byte) (Result, error) {
    var args []any
    if len(argsJSON) > 0 {
        if err := json.Unmarshal(argsJSON, &args); err != nil {
            var wrapper struct { Args []any `json:"args"` }
            if err2 := json.Unmarshal(argsJSON, &wrapper); err2 != nil { return Result{}, err }
            args = wrapper.Args
        }
    }

    fnVal := a.instance.Get(method)
    fn, ok := goja.AssertFunction(fnVal)
    if !ok { return Result{}, ErrMethodNotFound }

    jsArgs := make([]goja.Value, len(args))
    for i, arg := range args { jsArgs[i] = vm.ToValue(arg) }

    val, err := fn(a.instance, jsArgs...)
    if err != nil { return Result{}, err }

    payload, err := json.Marshal(val.Export())
    if err != nil { return Result{}, err }
    return Result{ValueJSON: payload}, nil
}
```

The public HTTP response for RPC should be:

```json
{ "ok": true, "result": 123 }
```

On errors:

```json
{ "ok": false, "error": { "code": "method_not_found", "message": "..." } }
```

### Fetch dispatch

Fetch dispatch should use the existing plain request/response idea from `gojahttp.RequestDTO` and `gojahttp.Response`. Do not implement WHATWG `Request`/`Response` yet.

```go
type FetchRequest struct {
    Method  string            `json:"method"`
    URL     string            `json:"url"`
    Path    string            `json:"path"`
    Query   map[string]any    `json:"query"`
    Headers map[string]string `json:"headers"`
    Body    any               `json:"body"`
    RawBody string            `json:"rawBody"`
}

type FetchResponse struct {
    Status  int               `json:"status"`
    Headers map[string]string `json:"headers,omitempty"`
    Body    any               `json:"body,omitempty"`
}
```

Actor pseudocode:

```go
func (a *Actor) callFetch(vm *goja.Runtime, req *FetchRequest) (Result, error) {
    fnVal := a.instance.Get("fetch")
    fn, ok := goja.AssertFunction(fnVal)
    if !ok { return Result{Response: &FetchResponse{Status: 404, Body: "fetch not implemented"}}, nil }

    val, err := fn(a.instance, vm.ToValue(req))
    if err != nil { return Result{}, err }

    var res FetchResponse
    if err := vm.ExportTo(val, &res); err != nil { return Result{}, err }
    if res.Status == 0 { res.Status = 200 }
    return Result{Response: &res}, nil
}
```

The gateway converts `FetchResponse` to `http.ResponseWriter`.

### Storage design

For MVP, use one SQLite file per object. It maps directly to Durable Objects' private-storage mental model and keeps query constraints simple.

Storage path:

```text
<storage-root>/<namespace>/<first-2-hash-bytes>/<hash>.sqlite
```

Example:

```text
var/durable-objects/COUNTER/ab/abcdef....sqlite
```

Create tables on open:

```sql
CREATE TABLE IF NOT EXISTS kv (
  key TEXT PRIMARY KEY,
  value_json BLOB NOT NULL,
  updated_at_ms INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS meta (
  key TEXT PRIMARY KEY,
  value_json BLOB NOT NULL,
  updated_at_ms INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS alarms (
  singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
  due_at_ms INTEGER NOT NULL,
  updated_at_ms INTEGER NOT NULL
);
```

Storage interface:

```go
type Storage interface {
    Get(ctx context.Context, key string) (any, bool, error)
    Put(ctx context.Context, key string, value any) error
    Delete(ctx context.Context, key string) (bool, error)
    List(ctx context.Context, prefix string, limit int) (map[string]any, error)
    Transaction(ctx context.Context, fn func(tx StorageTx) error) error
    SetAlarm(ctx context.Context, dueAt time.Time) error
    GetAlarm(ctx context.Context) (*time.Time, error)
    DeleteAlarm(ctx context.Context) error
    Close() error
}
```

JavaScript API:

```js
state.storage.get(key)
state.storage.put(key, value)
state.storage.delete(key)
state.storage.list(prefixOrOptions)
state.storage.transaction(fn)
state.storage.setAlarm(timestampMs)
state.storage.getAlarm()
state.storage.deleteAlarm()
```

Storage functions can be synchronous because the actor execution model is one request runs to completion. This gives deterministic behavior for simple read-modify-write flows:

```js
const current = state.storage.get("count") || 0;
state.storage.put("count", current + 1);
```

### Transactions

The MVP transaction API should be synchronous and non-reentrant:

```js
state.storage.transaction((tx) => {
  const current = tx.get("count") || 0;
  tx.put("count", current + 1);
});
```

Rules:

- `transaction(fn)` begins a SQLite transaction.
- `fn` runs synchronously on the actor owner thread.
- If `fn` throws, rollback and rethrow.
- If `fn` returns normally, commit.
- Do not allow async Promises from `fn` in MVP. If it returns a Promise, throw a clear error.

### Alarms

Each object can have one alarm. The alarm is durable metadata, not a Go timer stored only in memory.

Scheduler shape:

```text
AlarmScheduler tick
  │
  ├─ storage index finds due object IDs
  │
  ├─ Manager.Dispatch(KindAlarm, ObjectID)
  │
  └─ Actor calls instance.alarm()
```

For one-DB-per-object storage, scanning every SQLite file is simple but can become expensive. The MVP can maintain a small central alarm index in the storage root:

```sql
CREATE TABLE IF NOT EXISTS object_alarms (
  object_hash TEXT PRIMARY KEY,
  namespace TEXT NOT NULL,
  name TEXT NOT NULL,
  due_at_ms INTEGER NOT NULL
);
```

When `state.storage.setAlarm()` changes an object's local DB, also update the central index in the same Go storage method. If this dual write feels too risky, start with scanning local DB files and add the central index in Phase 2. For an intern implementation, I recommend central index from the beginning because it is small and keeps scheduler code straightforward.

Alarm dispatch pseudocode:

```go
func (s *AlarmScheduler) loop() {
    ticker := time.NewTicker(s.interval)
    defer ticker.Stop()
    for {
        select {
        case <-s.ctx.Done(): return
        case <-ticker.C:
            due, err := s.index.Due(s.ctx, time.Now(), s.batchSize)
            if err != nil { s.log.Error().Err(err).Msg("alarm scan failed"); continue }
            for _, item := range due {
                item := item
                go func() {
                    _, err := s.manager.Dispatch(s.ctx, Envelope{Kind: KindAlarm, ID: item.ID})
                    if err == nil { _ = s.index.Delete(s.ctx, item.ID) }
                }()
            }
        }
    }
}
```

Do not delete the alarm before successful dispatch. If the process crashes after delete and before JS execution, the alarm is lost.

### Gateway design

The gateway is a plain `http.Handler`:

```go
type Gateway struct {
    manager *Manager
    opts GatewayOptions
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    switch {
    case strings.HasPrefix(r.URL.Path, "/rpc/"):
        g.serveRPC(w, r)
    case strings.HasPrefix(r.URL.Path, "/fetch/"):
        g.serveFetch(w, r)
    default:
        http.NotFound(w, r)
    }
}
```

RPC route:

```text
POST /rpc/:namespace/:name/:method
```

Fetch route:

```text
ANY /fetch/:namespace/:name/*
```

Important parsing rule: object names may eventually contain slashes if URL-escaped, but MVP should keep names as one path segment and require clients to URL-escape special characters. Document this clearly.

RPC pseudocode:

```go
func (g *Gateway) serveRPC(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost { http.Error(w, "method not allowed", 405); return }
    namespace, name, method, err := parseRPCPath(r.URL.Path)
    if err != nil { writeError(w, err); return }

    id, err := NewObjectID(namespace, name)
    if err != nil { writeError(w, err); return }

    body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, g.opts.MaxBodyBytes))
    if err != nil { writeError(w, err); return }

    result, err := g.manager.Dispatch(r.Context(), Envelope{
        Kind: KindRPC, ID: id, Method: method, ArgsJSON: body,
    })
    if err != nil { writeError(w, err); return }
    writeRPCResult(w, result.ValueJSON)
}
```

Fetch pseudocode:

```go
func (g *Gateway) serveFetch(w http.ResponseWriter, r *http.Request) {
    namespace, name, rest, err := parseFetchPath(r.URL.Path)
    if err != nil { writeError(w, err); return }
    id, err := NewObjectID(namespace, name)
    if err != nil { writeError(w, err); return }

    req, err := NewFetchRequest(r, rest)
    if err != nil { writeError(w, err); return }

    result, err := g.manager.Dispatch(r.Context(), Envelope{Kind: KindFetch, ID: id, Request: req})
    if err != nil { writeError(w, err); return }
    writeFetchResponse(w, result.Response)
}
```

### Object stubs and `env`

Objects need to call other objects. The MVP should support synchronous JSON-only RPC through stubs.

JavaScript API:

```js
const counter = env.COUNTER.getByName("global");
const value = counter.rpc("increment", [1]);
```

Stub implementation:

```go
func newEnvObject(vm *goja.Runtime, manager *Manager, caller ObjectID) *goja.Object {
    env := vm.NewObject()
    for namespace := range manager.manifest.Objects {
        ns := vm.NewObject()
        _ = ns.Set("getByName", func(name string) *goja.Object {
            id, err := NewObjectID(namespace, name)
            if err != nil { panic(vm.NewGoError(err)) }
            return newStubObject(vm, manager, caller, id)
        })
        _ = env.Set(namespace, ns)
    }
    return env
}
```

Stub RPC must not pass `goja.Value` across actors. It must export to JSON-compatible Go values, dispatch through the manager, and import the result into the caller VM:

```go
_ = stub.Set("rpc", func(method string, args []any) goja.Value {
    payload, err := json.Marshal(args)
    if err != nil { panic(vm.NewGoError(err)) }

    res, err := manager.Dispatch(runtimebridge.CurrentOwnerContext(vm), Envelope{
        Kind: KindRPC, ID: targetID, Method: method, ArgsJSON: payload,
    })
    if err != nil { panic(vm.NewGoError(err)) }

    var out any
    if err := json.Unmarshal(res.ValueJSON, &out); err != nil { panic(vm.NewGoError(err)) }
    return vm.ToValue(out)
})
```

Be careful with deadlocks. If object A calls object B synchronously, and B calls A synchronously before A returns, A's owner is blocked waiting on B and cannot process B's call. The MVP should detect direct reentrant calls to the same `ObjectID` and return an error. Cross-object cycles are harder; document them as unsupported for synchronous MVP and prefer async stubs later.

### Resource control

Add request-level CPU/runtime budgets from the start.

At dispatch time:

```go
func (a *Actor) withInterrupt(ctx context.Context, d time.Duration, fn func() (Result, error)) (Result, error) {
    if d <= 0 { return fn() }
    timer := time.AfterFunc(d, func() {
        a.runtime.VM.Interrupt(fmt.Errorf("durable object CPU budget exceeded"))
    })
    defer func() {
        timer.Stop()
        a.runtime.VM.ClearInterrupt()
    }()
    return fn()
}
```

Caveats:

- `Runtime.Interrupt` interrupts JavaScript execution, not arbitrary blocking native Go calls.
- Storage methods should use context-aware SQL calls where possible.
- Do not present in-process goja as a hard sandbox for hostile code.

### Error model

Define typed errors with stable codes:

```go
type ErrorCode string

const (
    CodeBadRequest       ErrorCode = "bad_request"
    CodeUnknownNamespace ErrorCode = "unknown_namespace"
    CodeActorStartFailed ErrorCode = "actor_start_failed"
    CodeMethodNotFound   ErrorCode = "method_not_found"
    CodeStorageError     ErrorCode = "storage_error"
    CodeExecutionError   ErrorCode = "execution_error"
    CodeTimeout          ErrorCode = "timeout"
)
```

HTTP mapping:

| Code | HTTP status |
| --- | --- |
| `bad_request` | 400 |
| `unknown_namespace` | 404 |
| `method_not_found` | 404 |
| `timeout` | 504 |
| `execution_error` | 500 in dev includes message; production generic |
| `storage_error` | 500 |

## Decision records

### Decision: One `engine.Runtime` per live object

- **Context:** goja runtimes are not goroutine-safe and JavaScript values cannot safely move between runtimes. Durable Objects require per-object state isolation and predictable single-writer execution.
- **Options considered:** One runtime per process with many JS instances; one runtime per namespace; one runtime per live object.
- **Decision:** Use one `engine.Runtime` per live object.
- **Rationale:** This matches the Durable Objects mental model, reuses existing runtime lifecycle code, keeps object state isolated, and avoids concurrent VM access.
- **Consequences:** Startup and memory cost scale with live actors. Idle eviction and runtime pooling may become important later.
- **Status:** proposed.

### Decision: Use existing `RuntimeOwner` as the actor mailbox

- **Context:** The research source describes a mailbox goroutine per actor. The codebase already has `RuntimeOwner.Call()` and `Post()` backed by event loop scheduling.
- **Options considered:** Add a separate inbox channel in `Actor`; use `RuntimeOwner` directly; replace event loop scheduling.
- **Decision:** Use `RuntimeOwner` directly for MVP dispatch.
- **Rationale:** It already serializes VM work through `Scheduler.RunOnLoop`, propagates contexts, tracks active calls, and integrates with runtime shutdown.
- **Consequences:** Explicit queue metrics are less visible at first. If queue observability becomes necessary, wrap calls with manager-level counters before adding a second queue.
- **Status:** proposed.

### Decision: CommonJS bundle with `exports.objects` for MVP

- **Context:** The repository uses `goja_nodejs/require` and CommonJS modules. Full ES module support and Cloudflare-like syntax would add complexity.
- **Options considered:** CommonJS `exports.objects`; ES modules with `export class`; Cloudflare-compatible `DurableObject` base class.
- **Decision:** Start with CommonJS `exports.objects = { ClassName }`.
- **Rationale:** It fits existing runtime infrastructure and keeps bundle evaluation simple.
- **Consequences:** Authoring differs from Cloudflare Workers. A transpilation layer can later translate nicer syntax to this format.
- **Status:** proposed.

### Decision: Synchronous storage API first

- **Context:** Cloudflare storage is async, but supporting async correctly requires event-loop semantics, gates, and response-delaying behavior.
- **Options considered:** Promise-based storage immediately; synchronous KV storage; direct SQL only.
- **Decision:** Implement synchronous KV storage first.
- **Rationale:** One request runs to completion, and synchronous storage gives a simple data-race-free mental model.
- **Consequences:** Long storage operations block the actor. This is acceptable for MVP and can be revisited after correctness tests pass.
- **Status:** proposed.

### Decision: One SQLite database per object for MVP

- **Context:** Durable Objects have private object-local storage. SQLite is already available in the repository.
- **Options considered:** One DB per object; one DB per namespace; one sharded DB with object_id prefix; Postgres.
- **Decision:** Use one SQLite file per object initially.
- **Rationale:** It gives strong isolation, simple deletion/inspection, and easy mental mapping from object to storage.
- **Consequences:** Many objects create many files. A sharded storage backend should be a later interface-compatible implementation.
- **Status:** proposed.

### Decision: New gateway instead of overloading `gojahttp.Host`

- **Context:** `gojahttp.Host` routes into one JS runtime. Durable Objects must route to many runtimes by object identity.
- **Options considered:** Extend `gojahttp.Host`; mount object routes in Express; create a new `durableobjects.Gateway`.
- **Decision:** Create a new `Gateway` that is a normal `http.Handler`.
- **Rationale:** The routing target is a manager, not one runtime owner. Keeping it separate avoids confusing ownership semantics.
- **Consequences:** Some request DTO code should be reused or duplicated carefully. If duplication grows, extract shared request parsing later.
- **Status:** proposed.

### Decision: xgoja provider after core package

- **Context:** xgoja provider integration is needed for generated binaries, but core correctness should be testable without code generation.
- **Options considered:** Implement only as xgoja provider; implement only as a standalone package; core first then provider.
- **Decision:** Build `pkg/durableobjects` first, then wrap it in `pkg/xgoja/providers/durableobjects`.
- **Rationale:** Core tests can run quickly and deterministically. Provider work can follow established HTTP provider patterns.
- **Consequences:** Two phases are required. The provider should not contain core actor logic.
- **Status:** proposed.

## Implementation phases

### Phase 1: Core identity, manifest, storage, and manager skeleton

Files to create:

- `pkg/durableobjects/id.go`
- `pkg/durableobjects/manifest.go`
- `pkg/durableobjects/envelope.go`
- `pkg/durableobjects/storage.go`
- `pkg/durableobjects/storage_sqlite.go`
- `pkg/durableobjects/manager.go`

Acceptance tests:

- `NewObjectID("COUNTER", "global")` is stable and rejects empty inputs.
- Manifest resolves namespace to class.
- SQLite storage creates schema, stores JSON values, lists by prefix, deletes keys.
- Manager returns unknown namespace errors before starting actor.

### Phase 2: Actor runtime bootstrap and RPC dispatch

Files:

- `pkg/durableobjects/actor.go`
- `pkg/durableobjects/bundle.go`
- `pkg/durableobjects/modules.go`

Build a minimal counter test:

```go
func TestCounterRPCPersistsAcrossActorRestart(t *testing.T) {
    mgr := newTestManager(t, `
      class Counter {
        constructor(state, env) { this.state = state; this.env = env; }
        increment(by) {
          const current = this.state.storage.get("count") || 0;
          const next = current + (by || 1);
          this.state.storage.put("count", next);
          return next;
        }
      }
      exports.objects = { Counter };
    `)

    got := rpc(t, mgr, "COUNTER", "global", "increment", []any{1})
    require.Equal(t, float64(1), got)

    require.NoError(t, mgr.Evict(ctx, objectID("COUNTER", "global")))

    got = rpc(t, mgr, "COUNTER", "global", "increment", []any{1})
    require.Equal(t, float64(2), got)
}
```

### Phase 3: Gateway and fetch dispatch

Files:

- `pkg/durableobjects/gateway.go`
- `pkg/durableobjects/http_errors.go`

Tests:

- `POST /rpc/COUNTER/global/increment` returns JSON result.
- `GET /fetch/COUNTER/global/count` returns a plain response from `fetch(req)`.
- Bad routes return stable error JSON.
- Body size limits are enforced.

### Phase 4: Alarms

Files:

- `pkg/durableobjects/alarms.go`
- central alarm index implementation, if separate.

Tests:

- `state.storage.setAlarm(Date.now() + 10)` persists an alarm.
- Scheduler invokes `alarm()` and updates storage.
- Actor can be evicted before alarm and still wake on due alarm.
- Alarm is not removed if dispatch fails.

### Phase 5: Idle eviction and resource control

Add options:

```go
type Options struct {
    StorageRoot       string
    IdleTimeout       time.Duration
    EvictionInterval  time.Duration
    CPUTimeout        time.Duration
    MaxRequestBytes   int64
    DevErrors         bool
}
```

Tests:

- Actor is evicted after idle timeout.
- Active actor is not evicted.
- Infinite loop is interrupted and returns timeout error.
- Runtime closers close SQLite handles on eviction.

### Phase 6: xgoja provider integration

Files:

- `pkg/xgoja/providers/durableobjects/durableobjects.go`
- `pkg/xgoja/providers/durableobjects/config.go`
- `pkg/xgoja/providers/durableobjects/typescript.go`

Provider shape:

```go
const PackageID = "go-go-goja-durableobjects"
const HostServiceKey = "go-go-goja-durableobjects.manager"

type ExternalManagerService struct {
    Manager *durableobjects.Manager
    Gateway http.Handler
}
```

The provider should contribute:

- A module for object stubs if scripts need to call Durable Objects directly.
- Config sections for storage root, idle timeout, and manifest path.
- Optional command/provider integration for serving the gateway.
- TypeScript declarations for `state`, `env`, stubs, storage, and response shapes.

Follow the HTTP provider's registration style (`pkg/xgoja/providers/http/http.go:32-53`) and host-service lookup style (`pkg/xgoja/providers/http/http.go:124-168`).

## Testing strategy

### Unit tests

- `id_test.go`: validation, hashing, path-safe hash.
- `manifest_test.go`: parse YAML/JSON, unknown namespaces.
- `storage_sqlite_test.go`: schema creation, get/put/delete/list, transactions, alarm metadata, close behavior.
- `manager_test.go`: singleflight actor start, unknown namespace, eviction.
- `actor_test.go`: constructor called once per actor lifetime, method dispatch, method-not-found, JS exception mapping.
- `gateway_test.go`: route parsing, RPC/fetch response mapping.
- `alarms_test.go`: due scanning and retry behavior.

### Integration tests

- Counter persists across actor restart.
- Concurrent increments to the same actor produce sequential values without lost updates.
- Two names in the same namespace have isolated storage.
- Two namespaces with same object name have isolated storage.
- Object A can call object B through `env` stubs.
- Reentrant self-call is rejected with a clear error.

### Race tests

Run:

```bash
go test ./pkg/durableobjects -race -count=1
```

Important targets:

- `Manager` live actor map.
- Actor eviction while dispatch is in flight.
- Alarm dispatch while actor is idle or starting.
- Gateway requests during manager shutdown.

### Smoke test command

After implementation:

```bash
go test ./pkg/durableobjects ./pkg/xgoja/providers/durableobjects -count=1
```

If a command example is added:

```bash
go run ./cmd/goja-repl run examples/durableobjects/counter.js
```

Only add command-specific smoke tests once the actual command surface exists.

## Intern implementation notes and pitfalls

### Do not pass goja values between actors

Never store `goja.Value`, `*goja.Object`, or `goja.Callable` in `Manager` results or cross-actor envelopes. Convert to JSON-compatible Go values at the actor boundary. This protects runtime isolation and prevents accidental use of one VM's values in another VM.

### Do not hold manager locks while executing JavaScript

The manager lock protects the live actor map. It must not be held while calling `actor.Dispatch()`, starting runtime code, opening SQLite, or running JavaScript. Otherwise a slow actor can block unrelated objects.

### Be explicit about owner-thread-only fields

Fields like `Actor.instance` must be documented as owner-thread-only. If a test with `-race` catches access to these fields from outside `RuntimeOwner.Call()`, fix the design rather than suppressing the race.

### Synchronous cross-object calls can deadlock

Self-calls must be rejected. Cycles across objects can also deadlock if all calls are synchronous. For MVP, document synchronous stubs as best-effort for acyclic calls. A later async stub design can avoid this.

### Keep the API smaller than Cloudflare's

The first API is intentionally smaller:

- `exports.objects`, not `export class extends DurableObject`.
- plain fetch DTO, not WHATWG `Request`.
- sync storage, not async storage gates.
- one process, not global uniqueness.

This is a feature, not a weakness. It keeps implementation reviewable.

## Risks, alternatives, and open questions

### Risk: one runtime per live object may consume memory

This is expected. The MVP should rely on idle eviction. Add metrics for live actor count and runtime start/close count so the team can decide whether pooling or namespace-shared runtimes are worth exploring later.

### Risk: one SQLite file per object creates many files

This is acceptable for MVP demos and small local deployments. Keep storage behind an interface so a sharded backend can be introduced later without changing actor APIs.

### Risk: sync storage blocks actor execution

This is acceptable because the actor model intentionally serializes per-object work. If storage latency becomes a problem, add async storage with input/output gates as a separate design.

### Risk: JavaScript CPU interruption does not stop native Go calls

`goja.Runtime.Interrupt` only affects JS execution. Native storage calls must use contexts and SQLite timeouts. Do not claim hard tenant isolation.

### Open question: should manifest live in xgoja config or as a source asset?

For core tests, pass a `Manifest` directly. For xgoja, two options are viable:

1. YAML config section in `xgoja.yaml`.
2. Separate manifest file embedded as an asset.

The provider design should choose one once it is clear how generated applications will author Durable Objects.

### Open question: should alarms use central index in Phase 1?

I recommend yes, but it adds a second SQLite database. If implementation time is short, scan object DB files and add the index in Phase 2. The public API should not expose this choice.

### Open question: how much TypeScript support in MVP?

At minimum, provide declarations for `state`, `env`, storage, stubs, `FetchRequest`, and `FetchResponse`. Do not block core runtime work on perfect generated declarations.

## File reference map

Start reading in this order:

1. `pkg/engine/factory.go:184-260` — how an owned runtime is created.
2. `pkg/engine/runtime.go:32-127` — runtime fields and shutdown behavior.
3. `pkg/runtimeowner/types.go:9-27` — owner/scheduler interfaces.
4. `pkg/runtimeowner/runner.go:90-134` — serialized `Call()` implementation.
5. `pkg/runtimebridge/runtimebridge.go` — runtime service lookup and current owner context.
6. `pkg/gojahttp/host.go:94-163` — HTTP request to JavaScript callback pattern.
7. `pkg/gojahttp/request_response.go:17-73` — plain request DTO.
8. `modules/database/database.go:220-237` and `278-373` — module exports and SQL operations.
9. `modules/express/express.go:55-77` and `132-146` — runtime-aware module registration and JS-facing methods.
10. `pkg/xgoja/providers/http/http.go:32-168` — provider packaging and service injection.
11. `pkg/xgoja/app/host_services.go:30-45` and `164-188` — host service contribution and lookup.
12. `ttmp/2026/06/12/GOJA-DO-001--implement-durable-objects-for-go-go-goja/sources/01-durable-objects-research.md` — imported Durable Objects research source.

## Minimal demo target

The implementation is ready to show when this works:

```bash
curl -X POST http://127.0.0.1:8787/rpc/COUNTER/global/increment \
  -H 'content-type: application/json' \
  -d '[1]'
# {"ok":true,"result":1}

curl -X POST http://127.0.0.1:8787/rpc/COUNTER/global/increment \
  -H 'content-type: application/json' \
  -d '[1]'
# {"ok":true,"result":2}

# restart process

curl -X POST http://127.0.0.1:8787/rpc/COUNTER/global/increment \
  -H 'content-type: application/json' \
  -d '[1]'
# {"ok":true,"result":3}
```

That proves the durable-object kernel: stable identity, single-writer execution, private durable storage, restart recovery, and route-to-actor dispatch.

## References

- Imported source research: `sources/01-durable-objects-research.md`.
- Cloudflare Durable Objects overview: <https://developers.cloudflare.com/durable-objects/>.
- Cloudflare Durable Objects concepts: <https://developers.cloudflare.com/durable-objects/concepts/what-are-durable-objects/>.
- Cloudflare Durable Objects rules and gates: <https://developers.cloudflare.com/durable-objects/best-practices/rules-of-durable-objects/>.
- Cloudflare SQLite-backed Durable Object storage: <https://developers.cloudflare.com/durable-objects/api/sqlite-storage-api/>.
- goja upstream repository: <https://github.com/dop251/goja>.
