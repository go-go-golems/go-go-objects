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
- [ ] Unify provider configuration paths so public Glazed flags and static xgoja module config share one internal config/loader path.
- [ ] Add generated-binary xgoja fixture/example that embeds a Durable Objects bundle asset and exercises require("durableobjects").
- [ ] Implement or document automatic gateway mounting for generated HTTP hosts, including how Durable Objects /rpc and /fetch routes are exposed.
- [ ] Replace provider rpc/fetch context.Background dispatches with runtime/lifetime/current-owner context propagation.
- [ ] Add actor startup singleflight or equivalent duplicate-start suppression for concurrent first dispatches to the same object.
- [ ] Harden alarm persistence with crash reconciliation between object-local storage and the central alarm index.
- [ ] Add observability hooks: structured logs and basic metrics for actor startup, dispatch latency, errors, alarms, eviction, and storage failures.
- [ ] Define production storage/migration policy for SQLite files, schema versioning, backup expectations, and corrupted-object recovery.
- [ ] Audit security and resource limits: bundle trust model, CPU timeout behavior, request/body size limits, storage quotas, path/object-name validation, and error redaction defaults.
- [ ] Expand integration tests for HTTP gateway, alarms across restart, idle eviction, derived namespaces, explicit manifest aliases, and xgoja embedded assets.
- [ ] Add end-user examples and release documentation: authoring guide, CLI examples, xgoja YAML examples, embedded asset example, and known limitations.
- [ ] Run final release validation: go test ./... -count=1, race/concurrency-focused tests where feasible, docmgr doctor, clean git status, and tagged release notes.
