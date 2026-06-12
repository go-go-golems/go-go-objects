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
- [ ] Add an internal xgoja module config section for Durable Objects settings.
- [ ] Map public Glazed `durableobjects-*` flags into that internal module config.
- [ ] Keep filesystem mode and embedded asset mode using the same loader helpers.
- [ ] Avoid double-initializing managers when both module config and runtime initializers run.
- [ ] Update provider tests to exercise the standard `RuntimeFactory.NewRuntimeFromSections` path.

### Task 15: Generated embedded-asset fixture/example
- [ ] Add a checked-in example bundle and runtime spec or test fixture.
- [ ] Exercise `bundleAsset` without filesystem bundle dependencies.
- [ ] Verify `require("durableobjects").rpc(...)` works from the generated-style runtime path.
- [ ] Document how to declare the embedded asset in xgoja configuration.

### Task 16: Gateway mounting
- [ ] Decide whether Durable Objects should mount into the xgoja HTTP host automatically or via explicit host service wiring.
- [ ] Provide a shared HTTP host when both `go-go-goja-http` and Durable Objects are selected.
- [ ] Mount `/rpc` and `/fetch` without stripping the route prefixes expected by the Durable Objects gateway.
- [ ] Add a test proving a single xgoja HTTP host can serve Durable Objects gateway routes.
- [ ] Document fallback/manual mounting behavior.

### Task 17: Provider dispatch context propagation
- [ ] Replace `context.Background()` in provider `rpc`/`fetch` with a runtime-scoped context.
- [ ] Ensure dispatches cancel when the generated runtime closes.
- [ ] Keep JavaScript API synchronous for now while preserving cancellation.
- [ ] Add a regression test or code review note for the context source.

### Task 18: Actor startup duplicate suppression
- [ ] Add a per-object startup gate (`singleflight` or equivalent) around actor construction.
- [ ] Ensure concurrent first dispatches to the same object share one actor startup.
- [ ] Ensure failed startups do not poison future attempts.
- [ ] Add a concurrency test for simultaneous dispatches.

### Task 19: Alarm reconciliation
- [ ] Add a reconciler that compares central alarm index entries with object-local `alarm_at` metadata.
- [ ] Repair missing index rows for objects with local alarms.
- [ ] Remove stale index rows for objects without local alarms where feasible.
- [ ] Document remaining non-atomic cross-database behavior.
- [ ] Add restart/reconciliation tests.

### Task 20: Observability
- [ ] Add structured event hooks for actor start/stop, dispatch start/end, alarm dispatch, eviction, and storage errors.
- [ ] Add lightweight metrics counters/durations that tests can inspect.
- [ ] Keep default hooks no-op so embedders opt into collection.
- [ ] Document event names and fields.

### Task 21: Storage/migration policy
- [ ] Define SQLite schema version storage.
- [ ] Add migration hook or explicit version validation for object DBs and alarm index DB.
- [ ] Document backup, corruption, and recovery assumptions.
- [ ] Add README/operator notes for storage root layout.

### Task 22: Security/resource limits
- [ ] Audit object names and namespace validation against path traversal and unbounded path segments.
- [ ] Add gateway request/body size limits or document current limits.
- [ ] Default production gateway errors to redacted output.
- [ ] Document trusted-bundle model and CPU timeout behavior.
- [ ] Add tests for invalid names, oversized requests if implemented, and redaction.

### Task 23: Integration tests
- [ ] Add HTTP gateway RPC and fetch tests.
- [ ] Add alarm persistence/across-restart tests.
- [ ] Add idle eviction recovery tests.
- [ ] Add explicit manifest alias tests.
- [ ] Add xgoja embedded asset integration tests.

### Task 24: Examples and release documentation
- [ ] Add JavaScript authoring guide.
- [ ] Add CLI usage examples for demo and custom bundles.
- [ ] Add xgoja static config and embedded asset examples.
- [ ] Add known limitations and production-readiness notes.
- [ ] Update the design doc with final decisions.

### Task 25: Final release validation
- [ ] Run `go test ./... -count=1`.
- [ ] Run race/concurrency-focused tests where feasible.
- [ ] Run `docmgr doctor --ticket GOJA-DO-001 --stale-after 30`.
- [ ] Ensure `git status --short` is clean after final commits.
- [ ] Write release notes/tag guidance.
