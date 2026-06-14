---
Title: GOJA-DO-001 Tasks
Ticket: GOJA-DO-001
Status: active
Topics:
    - goja
    - architecture
    - durable-objects
    - actor-runtime
    - storage
DocType: task-list
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Task checklist for the Durable Objects design and implementation work.
LastUpdated: 2026-06-12T18:25:00-04:00
WhatFor: Track design completion and future implementation phases for GOJA-DO-001.
WhenToUse: Use while implementing or reviewing the Durable Objects work.
---

# Tasks

## Documentation and research

- [x] Create GOJA-DO-001 ticket workspace.
- [x] Import `/tmp/durable-objects.md` into `sources/` with docmgr frontmatter.
- [x] Inspect relevant runtime, HTTP, storage, module, and xgoja provider code.
- [x] Write the intern-facing architecture and implementation guide.
- [x] Write the investigation diary.
- [x] Implement Phase 1: core identity, manifest, storage, and manager skeleton.
- [x] Implement Phase 2: actor runtime bootstrap and RPC dispatch.
- [x] Implement Phase 3: HTTP gateway and fetch dispatch.
- [x] Implement Phase 4: alarms.
- [x] Implement Phase 5: idle eviction and resource controls.
- [x] Add explicit alarm scheduler and idle evictor wrappers.
- [x] Add built-in counter demo CLI.
- [x] Implement Phase 6: xgoja provider integration.
- [x] Unify provider configuration paths so public Glazed flags and static xgoja module config share one internal config/loader path.
- [x] Add generated-binary xgoja fixture/example that embeds a Durable Objects bundle asset and exercises require("durableobjects").
- [x] Implement or document automatic gateway mounting for generated HTTP hosts, including how Durable Objects /rpc and /fetch routes are exposed.
- [x] Replace provider rpc/fetch context.Background dispatches with runtime/lifetime/current-owner context propagation.
- [x] Add actor startup singleflight or equivalent duplicate-start suppression for concurrent first dispatches to the same object.
- [x] Harden alarm persistence with crash reconciliation between object-local storage and the central alarm index.
- [x] Add observability hooks: structured logs and basic metrics for actor startup, dispatch latency, errors, alarms, eviction, and storage failures.
- [x] Define production storage/migration policy for SQLite files, schema versioning, backup expectations, and corrupted-object recovery.
- [x] Audit security and resource limits: bundle trust model, CPU timeout behavior, request/body size limits, storage quotas, path/object-name validation, and error redaction defaults.
- [x] Expand integration tests for HTTP gateway, alarms across restart, idle eviction, derived namespaces, explicit manifest aliases, and xgoja embedded assets.
- [x] Add end-user examples and release documentation: authoring guide, CLI examples, xgoja YAML examples, embedded asset example, and known limitations.
- [x] Run final release validation: go test ./... -count=1, race/concurrency-focused tests where feasible, docmgr doctor, clean git status, and tagged release notes.


## Ship-readiness detailed checklist

### Task 14: Provider configuration unification
- [x] Add an internal xgoja module config section for Durable Objects settings.
- [x] Map public Glazed `durableobjects-*` flags into that internal module config.
- [x] Keep filesystem mode and embedded asset mode using the same loader helpers.
- [x] Avoid double-initializing managers when both module config and runtime initializers run.
- [x] Update provider tests to exercise the standard `RuntimeFactory.NewRuntimeFromSections` path.

### Task 15: Generated embedded-asset fixture/example
- [x] Add a checked-in example bundle and runtime spec or test fixture.
- [x] Exercise `bundleAsset` without filesystem bundle dependencies.
- [x] Verify `require("durableobjects").rpc(...)` works from the generated-style runtime path.
- [x] Document how to declare the embedded asset in xgoja configuration.

### Task 16: Gateway mounting
- [x] Decide whether Durable Objects should mount into the xgoja HTTP host automatically or via explicit host service wiring.
- [x] Provide a shared HTTP host when both `go-go-goja-http` and Durable Objects are selected.
- [x] Mount `/rpc` and `/fetch` without stripping the route prefixes expected by the Durable Objects gateway.
- [x] Add a test proving a single xgoja HTTP host can serve Durable Objects gateway routes.
- [x] Document fallback/manual mounting behavior.

### Task 17: Provider dispatch context propagation
- [x] Replace `context.Background()` in provider `rpc`/`fetch` with a runtime-scoped context.
- [x] Ensure dispatches cancel when the generated runtime closes.
- [x] Keep JavaScript API synchronous for now while preserving cancellation.
- [x] Add a regression test or code review note for the context source.

### Task 18: Actor startup duplicate suppression
- [x] Add a per-object startup gate (`singleflight` or equivalent) around actor construction.
- [x] Ensure concurrent first dispatches to the same object share one actor startup.
- [x] Ensure failed startups do not poison future attempts.
- [x] Add a concurrency test for simultaneous dispatches.

### Task 19: Alarm reconciliation
- [x] Add a reconciler that compares central alarm index entries with object-local `alarm_at` metadata.
- [x] Repair missing index rows for objects with local alarms.
- [x] Remove stale index rows for objects without local alarms where feasible.
- [x] Document remaining non-atomic cross-database behavior.
- [x] Add restart/reconciliation tests.

### Task 20: Observability
- [x] Add structured event hooks for actor start/stop, dispatch start/end, alarm dispatch, eviction, and storage errors.
- [x] Add lightweight metrics counters/durations that tests can inspect.
- [x] Keep default hooks no-op so embedders opt into collection.
- [x] Document event names and fields.

### Task 21: Storage/migration policy
- [x] Define SQLite schema version storage.
- [x] Add migration hook or explicit version validation for object DBs and alarm index DB.
- [x] Document backup, corruption, and recovery assumptions.
- [x] Add README/operator notes for storage root layout.

### Task 22: Security/resource limits
- [x] Audit object names and namespace validation against path traversal and unbounded path segments.
- [x] Add gateway request/body size limits or document current limits.
- [x] Default production gateway errors to redacted output.
- [x] Document trusted-bundle model and CPU timeout behavior.
- [x] Add tests for invalid names, oversized requests if implemented, and redaction.

### Task 23: Integration tests
- [x] Add HTTP gateway RPC and fetch tests.
- [x] Add alarm persistence/across-restart tests.
- [x] Add idle eviction recovery tests.
- [x] Add explicit manifest alias tests.
- [x] Add xgoja embedded asset integration tests.

### Task 24: Examples and release documentation
- [x] Add JavaScript authoring guide.
- [x] Add CLI usage examples for demo and custom bundles.
- [x] Add xgoja static config and embedded asset examples.
- [x] Add known limitations and production-readiness notes.
- [x] Update the design doc with final decisions.

### Task 25: Final release validation
- [x] Run `go test ./... -count=1`.
- [x] Run race/concurrency-focused tests where feasible.
- [x] Run `docmgr doctor --ticket GOJA-DO-001 --stale-after 30`.
- [x] Ensure `git status --short` is clean after final commits.
- [x] Write release notes/tag guidance.
- [x] Add Durable Objects xgoja command provider named serve, mounted as durableobjects serve by default.
- [x] Make durableobjects serve load filesystem bundles and generated embedded bundle assets through xgoja HostServices.
- [x] Factor reusable embeddable Durable Objects HTTP server helper for existing http.Server integrations.
- [x] Patch go-go-goja command provider context so provider commands receive HostServices and can resolve embedded assets.
- [x] Add xgoja custom template example for generating an embeddable Durable Objects HTTP runtime package instead of a main/command.
- [x] Add tests covering durableobjects serve command provider creation and server behavior.
- [x] Document xgoja durableobjects serve usage and custom template mode for existing http.Server applications.
- [x] Update Durable Objects xgoja integration for xgoja/v2 schema and PR75 mountable HTTP handler ABI.
- [x] Fix PR #1 code review comments and failing GitHub Actions.
