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
