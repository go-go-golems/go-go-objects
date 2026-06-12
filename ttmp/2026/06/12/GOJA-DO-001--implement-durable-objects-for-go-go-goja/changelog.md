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

