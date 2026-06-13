# Changelog

## 2026-06-12

- Initial workspace created


## 2026-06-12

Ticket created, research document imported, key codebase files related

### Related Files

- /tmp/durable-objects.md — Source research


## 2026-06-12

Senior research pass completed: wrote Durable Objects architecture and implementation guide, added investigation diary, and updated tasks

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-goja/ttmp/2026/06/12/GOJA-DO-001--implement-durable-objects-for-go-go-goja/design-doc/01-durable-objects-architecture-and-implementation-guide.md — Primary guide
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-goja/ttmp/2026/06/12/GOJA-DO-001--implement-durable-objects-for-go-go-goja/reference/01-investigation-diary.md — Diary
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-goja/ttmp/2026/06/12/GOJA-DO-001--implement-durable-objects-for-go-go-goja/tasks.md — Updated task checklist


## 2026-06-12

Uploaded GOJA DO 001 Durable Objects Guide bundle to reMarkable at /ai/2026/06/12/GOJA-DO-001

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-goja/ttmp/2026/06/12/GOJA-DO-001--implement-durable-objects-for-go-go-goja/design-doc/01-durable-objects-architecture-and-implementation-guide.md — Uploaded in reMarkable bundle


## 2026-06-12

Moved ticket into go-go-objects docmgr root and implemented initial durableobjects core: identity, manifest, SQLite storage, manager, actor RPC/fetch dispatch, gateway, and tests

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/durableobjects/actor.go — Actor implementation
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/durableobjects/gateway.go — Gateway implementation
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/durableobjects/manager.go — Manager implementation
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/durableobjects/storage_sqlite.go — Storage implementation


## 2026-06-12

Implemented alarms and idle eviction: central SQLite alarm index, due alarm dispatch, manager lifecycle context, actor activity tracking, and tests

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/durableobjects/alarms.go — Alarm index
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/durableobjects/durableobjects_test.go — Validation tests
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/durableobjects/manager.go — Alarm dispatch and idle eviction


## 2026-06-12

Added scheduler wrappers and runnable demo server: AlarmScheduler, IdleEvictor, active actor eviction test, cmd/go-go-objects, and README usage docs

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/README.md — Usage docs
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/cmd/go-go-objects/main.go — Demo server
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/durableobjects/scheduler.go — Scheduler wrappers


## 2026-06-12

Implemented xgoja provider integration: durableobjects provider package, config section, runtime initializer, JS rpc/fetch module, TypeScript descriptor, and provider tests

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/xgoja/providers/durableobjects/durableobjects.go — Provider implementation
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/xgoja/providers/durableobjects/durableobjects_test.go — Provider tests


## 2026-06-12

Added configurable CLI bundle and manifest loading with YAML/JSON manifest support, scheduler interval flags, README usage docs, and tests

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/cmd/go-go-objects/main.go — CLI implementation
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/cmd/go-go-objects/main_test.go — CLI tests


## 2026-06-12

Made manifests optional: namespaces now derive automatically from exports.objects keys using CamelCase to CAMEL_CASE conversion; CLI and xgoja provider accept bundle-only configuration

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/cmd/go-go-objects/main.go — Optional manifest CLI
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/durableobjects/bundle.go — Derived manifest
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/durableobjects/manifest.go — Namespace conversion
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/xgoja/providers/durableobjects/durableobjects.go — Optional manifest provider


## 2026-06-12

Added xgoja embedded asset support for Durable Objects provider module config; bundleAsset/manifestAsset can now load self-contained generated assets, while bundlePath/manifestPath remain supported for filesystem mode

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/README.md — Provider configuration docs
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/xgoja/providers/durableobjects/durableobjects.go — Embedded asset loading
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/xgoja/providers/durableobjects/durableobjects_test.go — Asset mode tests


## 2026-06-12

Added ship-readiness follow-up tasks covering provider config unification, generated embedded examples, gateway mounting, context propagation, singleflight startup, alarm reconciliation, observability, storage policy, security/resource limits, integration tests, documentation, and final release validation

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/ttmp/2026/06/12/GOJA-DO-001--implement-durable-objects-for-go-go-goja/tasks.md — New tasks 14-25


## 2026-06-12

Expanded tasks 14-25 into a detailed ship-readiness checklist with implementation and validation subtasks

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/ttmp/2026/06/12/GOJA-DO-001--implement-durable-objects-for-go-go-goja/tasks.md — Detailed ship-readiness checklist


## 2026-06-12

Worked through ship-readiness tasks 14-24: unified xgoja provider config via module config mapping, added embedded generated-style tests, mounted gateway on shared xgoja HTTP hosts, propagated runtime contexts, added duplicate actor startup suppression, reconciled alarm indexes, added observability events, added SQLite schema versioning, tightened ID validation, expanded tests, and wrote examples/release docs

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/README.md — Operations and examples docs
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/docs/release-notes.md — Release notes
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/durableobjects/alarms.go — Alarm reconciliation
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/durableobjects/manager.go — Runtime hardening
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/xgoja/providers/durableobjects/durableobjects.go — Provider hardening


## 2026-06-12

Completed final release validation for the ship-readiness pass: go test ./... -count=1, focused race tests for concurrency/provider paths, and docmgr doctor all passed

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/docs/release-notes.md — Validation command list
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/ttmp/2026/06/12/GOJA-DO-001--implement-durable-objects-for-go-go-goja/tasks.md — Task 25 checked


## 2026-06-12

Checked detailed ship-readiness subtasks after completing the hardening pass and final validation

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/ttmp/2026/06/12/GOJA-DO-001--implement-durable-objects-for-go-go-goja/tasks.md — Detailed subtasks 26-80 checked


## 2026-06-12

Fixed real CLI smoke-test bug: alarm scheduler now creates the SQLite storage root before opening alarms.sqlite, so a fresh --storage directory no longer logs repeated unable-to-open-database errors

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/durableobjects/alarms.go — Create alarm index storage root before sql.Open
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/durableobjects/durableobjects_test.go — Regression test for DispatchDueAlarms on missing storage root


## 2026-06-12

Added follow-up tasks for xgoja durableobjects serve command provider and embeddable HTTP server/template integration

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/ttmp/2026/06/12/GOJA-DO-001--implement-durable-objects-for-go-go-goja/tasks.md — Xgoja serve/template tasks


## 2026-06-12

Implemented missing xgoja UX pieces: Durable Objects serve command provider, reusable embeddable HTTP server helper, generated buildspec/template examples, and HostServices propagation into xgoja command providers

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-goja/pkg/xgoja/app/command_providers.go — HostServices propagation
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/examples/counter/xgoja-buildspec.yaml — Generated binary example with command provider
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/examples/templates/durableobjects_http_runtime.go.tmpl — Custom template mode example
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/durableobjects/server.go — Embeddable server helper
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/xgoja/providers/durableobjects/serve.go — Provider-owned serve command

