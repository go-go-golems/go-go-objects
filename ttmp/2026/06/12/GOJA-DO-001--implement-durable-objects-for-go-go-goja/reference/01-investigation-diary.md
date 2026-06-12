---
Title: Investigation Diary
Ticket: GOJA-DO-001
Status: active
Topics:
    - goja
    - architecture
    - durable-objects
    - actor-runtime
    - storage
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: README.md
      Note: |-
        Step 5 project usage docs
        Step 7 CLI usage documentation
    - Path: cmd/go-go-objects/main.go
      Note: |-
        Step 5 demo server
        Step 7 configurable CLI loading
    - Path: cmd/go-go-objects/main_test.go
      Note: Step 7 CLI loading tests
    - Path: modules/database/database.go
      Note: SQLite module evidence for storage design.
    - Path: pkg/durableobjects/actor.go
      Note: Implementation Step 3 actor work
    - Path: pkg/durableobjects/alarms.go
      Note: Step 4 alarm index implementation
    - Path: pkg/durableobjects/durableobjects_test.go
      Note: |-
        Validation tests for Step 3
        Step 4 tests
    - Path: pkg/durableobjects/manager.go
      Note: |-
        Implementation Step 3 manager work
        Step 4 alarm dispatch
    - Path: pkg/durableobjects/scheduler.go
      Note: Step 5 scheduler wrappers
    - Path: pkg/engine/factory.go
      Note: Runtime factory evidence for actor runtime design.
    - Path: pkg/gojahttp/host.go
      Note: HTTP dispatch evidence for gateway design.
    - Path: pkg/runtimeowner/runner.go
      Note: Runtime owner evidence for actor mailbox design.
    - Path: pkg/xgoja/providers/durableobjects/durableobjects.go
      Note: Step 6 xgoja provider implementation
    - Path: pkg/xgoja/providers/durableobjects/durableobjects_test.go
      Note: Step 6 provider tests
    - Path: ttmp/2026/06/12/GOJA-DO-001--implement-durable-objects-for-go-go-goja/design-doc/01-durable-objects-architecture-and-implementation-guide.md
      Note: Final implementation guide produced from this investigation.
ExternalSources: []
Summary: Chronological investigation diary for the Durable Objects architecture and implementation guide.
LastUpdated: 2026-06-12T16:40:00-04:00
WhatFor: Use this to understand how the GOJA-DO-001 design guide was researched and produced.
WhenToUse: Read when resuming or reviewing the Durable Objects analysis work.
---






# Diary

## Goal

This diary records the research and documentation work for GOJA-DO-001, whose goal is to design a go-go-goja implementation of Durable Objects.

## Step 1: Create the ticket and import the source research

The first step established the docmgr workspace and preserved the user's existing Durable Objects research as ticket source material. This matters because the design work should build from the prior research instead of rediscovering the architecture from scratch.

The imported research frames the feature as an addressable actor runtime, not a Cloudflare Workers clone. That framing became the central constraint for the implementation guide.

### Prompt Context

**User prompt (verbatim):** "Create a new docmgr ticket to implement a go-go-goja version of durable objecst. I did some research abotu a pure goja version of it in /tmp/durable-objects.md, import the file into the ticket's source/ directory and read it carefully. Then inspect the codebase and find and study all the relevant document you can find. 

Do not write a design or analysis document yet, instead focus on studying and tell me what you found (in the chat) when done. We'll then delegate ato the senior researcher to finish doing the analysis."

**Assistant interpretation:** Create a ticket, import the research source, study code and documents, but only report findings in chat.

**Inferred user intent:** Prepare a clean research handoff for a later detailed design pass.

**Commit (code):** N/A — documentation-only workspace setup.

### What I did
- Created ticket `GOJA-DO-001`.
- Copied `/tmp/durable-objects.md` to `sources/01-durable-objects-research.md`.
- Added frontmatter to make the imported source valid docmgr markdown.
- Related important runtime, HTTP, database, module, and provider files to the ticket.

### Why
- The source material needed to become a stable ticket artifact.
- The later implementation guide needed file-backed evidence.

### What worked
- Ticket creation succeeded.
- Source import succeeded after adding valid frontmatter.
- Core related files were identified quickly.

### What didn't work
- Initial `docmgr doc relate` failed because the imported source file did not have frontmatter:
  `Error: document has invalid frontmatter ... frontmatter delimiters '---' not found`.
- The fix was to prepend docmgr frontmatter and rerun the relation command.

### What I learned
- The source research already contains a strong MVP boundary: one process, one bundle, one runtime per actor, sync KV, alarms, no cluster, no Worker compatibility.
- The repository already has runtime ownership, HTTP dispatch, SQLite, and xgoja provider patterns.

### What was tricky to build
- The imported research was a plain markdown note, not a docmgr document. The underlying issue was metadata validation, not content. I fixed it by adding frontmatter rather than changing the research body.

### What warrants a second pair of eyes
- Ensure the imported source remains faithful to `/tmp/durable-objects.md` except for frontmatter.

### What should be done in the future
- Keep source research separate from the design doc so the design can evolve without losing provenance.

### Code review instructions
- Start with `sources/01-durable-objects-research.md`.
- Validate with `docmgr validate frontmatter --doc <source-doc> --suggest-fixes` if needed.

### Technical details
- Ticket path: `ttmp/2026/06/12/GOJA-DO-001--implement-durable-objects-for-go-go-goja`.
- Source path: `sources/01-durable-objects-research.md`.

## Step 2: Senior research pass and implementation guide

The second step performed the requested senior-researcher pass. I reviewed the prior findings, re-inspected the key runtime files with line-number evidence, and wrote an intern-facing architecture and implementation guide.

The resulting guide recommends a small Durable Objects kernel in `pkg/durableobjects`, followed by xgoja provider integration. The central design is one `engine.Runtime` per live object, with existing `RuntimeOwner.Call()` serving as the actor mailbox.

### Prompt Context

**User prompt (verbatim):** "You are now the senior researcher. Review and do your own investigation if necessary, then Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Produce the primary design/implementation guide in the ticket and upload it to reMarkable.

**Inferred user intent:** Give a future intern and reviewer a clear, evidence-backed implementation roadmap for Durable Objects.

**Commit (code):** N/A — documentation-only analysis.

### What I did
- Read the ticket research source.
- Re-inspected runtime, owner, HTTP, database, native module, timer, and xgoja provider code.
- Created `design-doc/01-durable-objects-architecture-and-implementation-guide.md`.
- Wrote architecture, decisions, APIs, pseudocode, implementation phases, tests, and file references.
- Created this investigation diary.

### Why
- The prior chat findings were useful but not sufficient as a durable intern-facing design package.
- The implementation touches multiple subsystems, so the document needed to explain the mental model and current code evidence.

### What worked
- The existing codebase strongly supports the proposed design:
  - `engine.Runtime` already models an owned VM lifecycle.
  - `runtimeowner.RuntimeOwner` already serializes JS work through an event-loop scheduler.
  - `gojahttp.Host` demonstrates Go HTTP to JS callback dispatch.
  - `modules/database` provides SQLite and transaction patterns.
  - `pkg/xgoja/providers/http` provides a provider packaging pattern.

### What didn't work
- My first attempt to collect line-number evidence used paths relative to the workspace root, but the target files live under the `go-go-goja/` subdirectory. The commands failed with messages like `nl: pkg/engine/factory.go: No such file or directory`.
- I reran the commands from `/home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-goja` and collected the needed evidence.

### What I learned
- The correct actor mailbox abstraction is likely not a new explicit channel. `RuntimeOwner.Call()` already provides the serialization boundary over the event loop.
- The main design tension is actor-specific module injection: an actor needs `state` and `env` bound to its storage and manager, while `RuntimeFactory` is immutable. The guide recommends per-actor factory construction for MVP and postpones engine overlay changes until profiling demands them.

### What was tricky to build
- The main subtlety is avoiding two queues. The research source describes a mailbox goroutine, but the repository already has event-loop scheduling and runtime ownership. Adding a second queue would create more lifecycle and observability complexity without improving MVP correctness. The guide explicitly chooses `RuntimeOwner` as the actor mailbox.
- Another subtlety is cross-object calls. Synchronous stubs can deadlock on self-calls or cycles. The guide recommends detecting self-calls and documenting cyclic synchronous calls as unsupported until async stubs exist.

### What warrants a second pair of eyes
- The per-actor factory construction decision should be reviewed for startup overhead.
- The central alarm index versus scanning object DB files should be reviewed before implementation.
- The synchronous cross-object stub limitations should be reviewed before promising object-to-object RPC in user docs.

### What should be done in the future
- Implement the core package in phases before provider integration.
- Add TypeScript declarations once the API is stable.
- Revisit async storage and output gates only after the synchronous MVP is tested.

### Code review instructions
- Start with `design-doc/01-durable-objects-architecture-and-implementation-guide.md`.
- Then inspect the referenced files in the order listed in its File reference map.
- Validate the documentation with `docmgr doctor --ticket GOJA-DO-001 --stale-after 30`.

### Technical details
- Primary guide: `design-doc/01-durable-objects-architecture-and-implementation-guide.md`.
- Key implementation package proposed: `pkg/durableobjects`.
- Key provider package proposed: `pkg/xgoja/providers/durableobjects`.

## Step 3: Move ticket to go-go-objects and start implementation

This step moved the GOJA-DO-001 ticket from the old `go-go-goja/ttmp` docmgr root into the new `go-go-objects/ttmp` root, then started the implementation in the new `go-go-objects` repository. The first code slice implements the local Durable Objects kernel rather than xgoja provider integration: identity, manifest validation, SQLite storage, runtime-backed actors, manager dispatch, state/env JS objects, HTTP gateway, and tests.

The implementation intentionally follows the guide's MVP shape. Each live object owns an `engine.Runtime`; `RuntimeOwner.Call()` serializes JavaScript dispatch; storage is a private SQLite file per object; RPC and fetch values cross the Go/JS boundary through JSON-compatible Go values rather than `goja.Value` sharing.

### Prompt Context

**User prompt (verbatim):** "I ctually created ./go-go-objects/ and pointed docmgr at it. use `mv` to move the current ticket over, and then start implementing the projects."

**Assistant interpretation:** Move the existing GOJA-DO-001 ticket into the new go-go-objects doc tree and begin implementing the Durable Objects project in that repository.

**Inferred user intent:** Stop treating this as go-go-goja documentation-only work and begin the standalone go-go-objects implementation.

**Commit (code):** N/A — changes are uncommitted in the working tree.

### What I did
- Moved `go-go-goja/ttmp/2026/06/12/GOJA-DO-001--implement-durable-objects-for-go-go-goja` to `go-go-objects/ttmp/2026/06/12/` using `mv`.
- Updated the workspace `.ttmp.yaml` to point docmgr at `go-go-objects/ttmp` and its vocabulary file.
- Renamed the Go module from `github.com/go-go-golems/XXX` to `github.com/go-go-golems/go-go-objects`.
- Added a local `replace github.com/go-go-golems/go-go-goja => ../go-go-goja` so the new project can reuse `engine.Runtime` and `runtimebridge` during workspace development.
- Added `pkg/durableobjects` with initial core files:
  - `id.go`
  - `manifest.go`
  - `envelope.go`
  - `errors.go`
  - `storage.go`
  - `storage_sqlite.go`
  - `bundle.go`
  - `manager.go`
  - `actor.go`
  - `modules.go`
  - `gateway.go`
  - `durableobjects_test.go`
- Updated `tasks.md` to mark Phases 1, 2, and 3 complete for the initial implementation slice.
- Ran `gofmt`, `go mod tidy`, `go test ./pkg/durableobjects -count=1`, and `go test ./... -count=1`.

### Why
- The user's new repository is `go-go-objects`; docs and implementation should live there.
- A small working kernel is more useful than another design pass.
- Starting with tests around counter persistence and gateway dispatch validates the design's most important invariants.

### What worked
- `docmgr status --summary-only` now reports the root as `go-go-objects/ttmp`.
- `go test ./pkg/durableobjects -count=1` passes.
- `go test ./... -count=1` passes.
- Counter RPC persists across actor eviction through SQLite-backed per-object storage.
- `/rpc/COUNTER/global/increment` and `/fetch/COUNTER/global/count` gateway flows are covered by tests.

### What didn't work
- The first `go test` failed because `github.com/go-go-golems/go-go-goja v0.0.0` was treated as a remote version: `unknown revision v0.0.0`. I added a local `replace` directive to `../go-go-goja`.
- The first fetch gateway test returned an empty body because Go struct fields were exposed to JavaScript as capitalized properties instead of lower-case `path`, `method`, etc. I changed actor fetch dispatch to pass an explicit map with lower-case keys.
- A later `go test ./...` failed because `pkg/doc.go` used package `objects` while generated `pkg/logcopter.go` used package `pkg`: `found packages objects (doc.go) and pkg (logcopter.go)`. I changed `pkg/doc.go` to package `pkg`.

### What I learned
- The runtime reuse path works: `go-go-objects` can build actors on top of `go-go-goja/pkg/engine` without modifying go-go-goja.
- Passing Go structs directly into goja is not the right default for public JavaScript DTOs when lower-case property names matter; explicit maps avoid ambiguity.
- The scaffold still contains placeholder names (`cmd/XXX`, logcopter package name). Those are separate cleanup tasks.

### What was tricky to build
- The main implementation edge was translating values at runtime boundaries. Actor RPC exports JavaScript return values to JSON and gateway responses write that JSON. Fetch dispatch needed an explicit lower-case map because JavaScript code expects `req.path`, not `req.Path`.
- Another edge was runtime lifetime context. Actor creation initially used the request context as the runtime lifetime parent, which would cancel actor runtimes when the startup request ended. I changed actor runtime creation to use `context.Background()` as the lifetime parent for now; a future manager-level lifecycle context should replace that.

### What warrants a second pair of eyes
- The `Manager` start path currently has a simple duplicate-start avoidance pattern, not a full singleflight start gate. Concurrent first requests may do extra work and close one duplicate actor.
- Actor lifetime currently uses `context.Background()`; the manager should eventually own a lifecycle context.
- Gateway route parsing currently treats object names as one path segment. That should be documented or expanded with URL escaping rules.
- The local `replace` directive is useful in this workspace but should be reviewed before publishing the module.

### What should be done in the future
- Implement Phase 4 alarms: scheduler, due scanning, and wake-from-evicted tests.
- Implement Phase 5 idle eviction and stronger resource controls.
- Rename scaffold leftovers (`cmd/XXX`, logcopter package name) when the CLI shape is known.
- Add xgoja provider integration after core behavior stabilizes.

### Code review instructions
- Start with `pkg/durableobjects/manager.go` and `pkg/durableobjects/actor.go`.
- Then review `pkg/durableobjects/storage_sqlite.go` for schema and JSON encoding.
- Then review `pkg/durableobjects/modules.go` for JavaScript-facing `state.storage` and `env` behavior.
- Validate with `go test ./... -count=1` from `go-go-objects`.

### Technical details
- Passing test command: `cd go-go-objects && go test ./... -count=1`.
- Main package implemented: `github.com/go-go-golems/go-go-objects/pkg/durableobjects`.
- Ticket now lives under `go-go-objects/ttmp/2026/06/12/GOJA-DO-001--implement-durable-objects-for-go-go-goja`.

## Step 4: Add alarms, idle eviction, and manager lifecycle context

This step continued the implementation by adding the next pieces of the Durable Objects kernel: persistent alarm indexing, due-alarm dispatch, idle actor eviction, and a manager-owned lifecycle context for actor runtimes. The result is still intentionally small, but it now covers the core behavior needed for objects to wake after eviction and for managers to close all owned runtimes cleanly.

The implementation keeps the alarm scheduler explicit for now: callers invoke `Manager.DispatchDueAlarms(ctx, now, limit)`. This avoids introducing background goroutine lifecycle complexity before the core semantics are tested. A periodic scheduler can later wrap this method.

### Prompt Context

**User prompt (verbatim):** "continue."

**Assistant interpretation:** Continue implementing the Durable Objects runtime from the previous phase.

**Inferred user intent:** Advance beyond the initial RPC/fetch/storage kernel into the next phases from the implementation guide.

**Commit (code):** N/A — changes are uncommitted in the working tree.

### What I did
- Added `pkg/durableobjects/alarms.go`.
- Added a central SQLite alarm index at `<storage-root>/alarms.sqlite` for due-alarm lookup.
- Updated `SQLiteStorage.SetAlarm` and `DeleteAlarm` to keep the object-local alarm table and central alarm index in sync.
- Added `Manager.DispatchDueAlarms(ctx, now, limit)`.
- Added manager-owned lifecycle context and cancellation in `Manager.Close`.
- Changed actor runtimes to use the manager lifecycle context instead of `context.Background()`.
- Added actor activity tracking with `active` and `lastUsedNS` atomics.
- Added `Manager.EvictIdle(ctx, now)` and actor idle detection.
- Added tests for waking an evicted actor through an alarm and for idle eviction preserving durable state.
- Ran `gofmt` and `go test ./... -count=1`.

### Why
- Alarms are part of the MVP described in the research and design guide.
- Idle eviction is required to make one-runtime-per-live-object practical.
- Manager-owned lifecycle context fixes the earlier placeholder runtime lifetime behavior.

### What worked
- Due alarms can wake an actor after it has been evicted.
- Alarm dispatch deletes the central alarm index entry after successful dispatch.
- Idle eviction closes an actor runtime and the next RPC recreates the actor with preserved SQLite state.
- `go test ./... -count=1` passes.

### What didn't work
- No major failures in this step. The implementation compiled and passed after adding the new code and tests.

### What I learned
- Keeping `DispatchDueAlarms` as a direct method makes alarm semantics easy to test. A background scheduler should be a thin wrapper around it.
- The actor activity tracker is enough for initial idle eviction, but future long-running request tests should verify active actors are never evicted.

### What was tricky to build
- The subtle part is keeping two alarm records coherent: the object-local table and the central index. For now, `SetAlarm` writes local storage first and then the central index; `DeleteAlarm` deletes local first and then the index. This is not a cross-database atomic transaction, but it is acceptable for the MVP and should be documented as a future hardening point.
- Idle eviction needed a careful lock pattern: collect candidates under the manager lock, remove matching actors from the map, release the lock, then close runtimes outside the lock.

### What warrants a second pair of eyes
- Alarm index consistency across crashes should be reviewed before production use.
- `DispatchDueAlarms` currently stops on first dispatch/delete error. A batch mode that records per-alarm errors may be better later.
- Idle eviction has no background ticker yet; callers must invoke `EvictIdle` manually.

### What should be done in the future
- Add a background `AlarmScheduler` wrapper with explicit `Start`/`Stop` lifecycle.
- Add a background idle evictor wrapper if desired.
- Add tests ensuring active actors are not evicted.
- Add xgoja provider integration.

### Code review instructions
- Start with `pkg/durableobjects/alarms.go` for the alarm index.
- Then inspect `SQLiteStorage.SetAlarm` / `DeleteAlarm` in `storage_sqlite.go`.
- Review `Manager.DispatchDueAlarms` and `Manager.EvictIdle` in `manager.go`.
- Validate with `go test ./... -count=1`.

### Technical details
- Passing test command: `cd go-go-objects && go test ./... -count=1`.
- Alarm index path: `<storage-root>/alarms.sqlite`.
- Due alarm method: `Manager.DispatchDueAlarms(ctx, now, limit)`.
- Idle eviction method: `Manager.EvictIdle(ctx, now)`.

## Step 5: Add maintenance wrappers and demo server

This step made the runtime easier to operate by adding explicit background-loop wrappers around the alarm and idle-eviction methods, then replaced the scaffold command with a small runnable counter demo server. The core manager methods remain directly testable, while the wrappers provide the shape applications need for long-running processes.

The demo server is intentionally minimal. It embeds a Counter Durable Object class, mounts the Durable Objects gateway directly as the HTTP handler, starts the alarm scheduler and idle evictor, and exposes the same `/rpc` and `/fetch` endpoints used in tests.

### Prompt Context

**User prompt (verbatim):** (same as Step 4)

**Assistant interpretation:** Continue implementing the Durable Objects runtime and make the project more runnable.

**Inferred user intent:** Move from core library behavior toward an operable project skeleton.

**Commit (code):** N/A — changes are uncommitted in the working tree.

### What I did
- Added `pkg/durableobjects/scheduler.go`.
- Added `AlarmScheduler` with `Tick` and `Run` methods.
- Added `IdleEvictor` with `Tick` and `Run` methods.
- Added `Manager.ActiveCount()` for observability and tests.
- Added tests for scheduler tick, idle evictor tick, and not evicting an active actor.
- Renamed scaffold command directory from `cmd/XXX` to `cmd/go-go-objects`.
- Implemented a built-in counter demo server in `cmd/go-go-objects/main.go`.
- Rewrote `README.md` with project overview, capabilities, demo commands, JavaScript authoring model, and development command.
- Ran `gofmt` and `go test ./... -count=1`.

### Why
- `DispatchDueAlarms` and `EvictIdle` are good primitive methods, but applications need repeatable loop wrappers.
- The repository scaffold still had placeholder command and README content; a demo server validates and explains the runtime better.

### What worked
- `AlarmScheduler.Tick` dispatches due alarms through the manager.
- `IdleEvictor.Tick` evicts idle actors through the manager.
- Active actors are protected from idle eviction while a long-running dispatch is in progress.
- The demo command compiles as part of `go test ./...`.

### What didn't work
- No major failures in this step. The first implementation passed after formatting and tests.

### What I learned
- Keeping wrappers thin is the right shape: tests can exercise `Tick`, while applications can use `Run` with a context.
- The active-dispatch counter is useful not only for eviction correctness but also as a future metric hook.

### What was tricky to build
- The active-actor eviction test needed a JavaScript method that keeps the actor busy long enough for the test to attempt eviction. The test bundle now includes a small `spin(ms)` method using `Date.now()`.
- The scheduler `Run` methods should not crash the loop on one alarm/eviction error. They accept an optional `ErrorHandler` and continue ticking.

### What warrants a second pair of eyes
- The demo server currently uses a hard-coded Counter bundle. A future CLI should accept a bundle path and manifest path.
- The scheduler wrappers continue after errors. That is good for availability, but production code may need metrics and backoff.
- `Manager.ActiveCount()` reports live actors only, not in-flight start attempts.

### What should be done in the future
- Add configurable bundle/manifest loading to the CLI.
- Add xgoja provider integration.
- Add metrics/logging around scheduler ticks, dispatched alarms, evictions, actor starts, and actor closes.

### Code review instructions
- Review `pkg/durableobjects/scheduler.go` for loop semantics.
- Review `pkg/durableobjects/durableobjects_test.go` tests around scheduler/evictor behavior.
- Review `cmd/go-go-objects/main.go` for demo server lifecycle and endpoints.
- Validate with `go test ./... -count=1`.

### Technical details
- Demo command: `go run ./cmd/go-go-objects --addr 127.0.0.1:8787 --storage ./var/durable-objects`.
- Alarm loop wrapper: `NewAlarmScheduler(mgr, interval, limit, onError).Run(ctx)`.
- Idle loop wrapper: `NewIdleEvictor(mgr, interval, onError).Run(ctx)`.

## Step 6: Add xgoja provider integration

This step added the first xgoja provider layer for `go-go-objects`. The provider is thin glue over `pkg/durableobjects`: it registers a `durableobjects` CommonJS module, exposes a Glazed config section, initializes a `durableobjects.Manager` from bundle and manifest files, starts optional alarm/idle loops, and closes the manager with the owning engine runtime.

The provider deliberately keeps the Durable Objects runtime implementation in `pkg/durableobjects`. xgoja-specific code lives under `pkg/xgoja/providers/durableobjects`, mirroring the existing go-go-goja HTTP provider pattern while avoiding a dependency from core runtime code back into xgoja.

### Prompt Context

**User prompt (verbatim):** "go ahead."

**Assistant interpretation:** Implement the xgoja provider integration described in the previous answer.

**Inferred user intent:** Make the Durable Objects runtime selectable and usable from generated xgoja runtimes, not only from direct Go code or the demo server.

**Commit (code):** N/A — changes are uncommitted in the working tree.

### What I did
- Added `pkg/xgoja/providers/durableobjects/durableobjects.go`.
- Added provider package ID `go-go-objects-durableobjects`.
- Registered a `durableobjects` module with TypeScript declarations.
- Added config section `durableobjects` with fields:
  - `enabled`
  - `storage-root`
  - `bundle-path`
  - `manifest-path`
  - `cpu-timeout`
  - `idle-timeout`
  - `alarm-interval`
  - `idle-interval`
- Implemented `RuntimeInitializerCapability` that loads manifest and bundle files, creates a `durableobjects.Manager`, starts optional scheduler loops, and registers cleanup with the engine runtime.
- Implemented JS module exports:
  - `rpc(namespace, name, method, args)`
  - `fetch(namespace, name, request)`
- Added `GatewayService` and `HostServiceKey` for externally supplied manager/gateway services.
- Added `pkg/xgoja/providers/durableobjects/durableobjects_test.go`.
- Ran `gofmt`, `go mod tidy`, and `go test ./... -count=1`.

### Why
- Generated xgoja binaries need provider packages to select modules, expose config, and initialize runtime-owned Go services.
- The provider should make Durable Objects available to xgoja scripts while preserving the direct Go API.

### What worked
- Provider registration resolves the `durableobjects` module and capability.
- Config section is exposed with the expected slug and prefix.
- Runtime initialization creates a manager from temporary bundle/manifest files in tests.
- xgoja JS can call `require("durableobjects").rpc("COUNTER", "global", "increment", [2])` and dispatch into the Durable Objects manager.
- `go test ./... -count=1` passes.

### What didn't work
- No major failures in this step. I initially considered implementing `HostServiceContributionCapability`, but removed the no-op implementation because the current provider initializes the manager at runtime-initializer time rather than pre-runtime host-service contribution time.

### What I learned
- The existing xgoja provider APIs are enough for a thin integration without modifying go-go-goja.
- The right first provider surface is a JS module plus runtime initializer. Mounting the gateway into an outer Go HTTP mux can be a follow-up once a concrete generated host application needs it.

### What was tricky to build
- Provider initialization happens after the engine runtime exists, while host services are contributed before runtime construction. Because the manager is currently created in `InitRuntimeFromSections`, the provider cannot also contribute a fully initialized gateway host service through `HostServiceContributionCapability` without changing the construction flow. The current code exposes `GatewayService` for externally supplied managers but does not yet auto-contribute a gateway service.
- The provider starts background loops from the engine runtime context and cancels them through a registered closer. This keeps loop lifetime tied to the generated runtime.

### What warrants a second pair of eyes
- Whether the provider should eventually construct the manager earlier as a host service so embedding Go applications can mount the gateway more directly.
- Whether `rpc`/`fetch` should use the current owner context instead of `context.Background()` for dispatch cancellation.
- Whether provider config should support embedded assets instead of only filesystem bundle/manifest paths.

### What should be done in the future
- Add provider support for embedded bundle and manifest sources.
- Add a command or host-service integration for mounting the Durable Objects gateway in generated web servers.
- Expand TypeScript declarations once the JS API stabilizes.

### Code review instructions
- Start with `pkg/xgoja/providers/durableobjects/durableobjects.go`.
- Review tests in `pkg/xgoja/providers/durableobjects/durableobjects_test.go`.
- Validate with `go test ./... -count=1`.

### Technical details
- Provider package ID: `go-go-objects-durableobjects`.
- Module name: `durableobjects`.
- Config section slug: `durableobjects`.
- Required enabled config: `bundle-path`, `manifest-path`, and `storage-root`.

## Step 7: Add configurable CLI bundle and manifest loading

This step converted the demo CLI from a built-in-only counter server into a reusable local Durable Objects host. The built-in counter remains the default for a zero-configuration smoke test, but callers can now provide their own CommonJS bundle and JSON/YAML manifest with `--bundle` and `--manifest`.

The new CLI flags also expose the same runtime maintenance knobs used by the library: CPU timeout, idle timeout, alarm scheduler interval, idle eviction interval, and development error verbosity.

### Prompt Context

**User prompt (verbatim):** "commit at appropriate intervals in the fturue."

**Assistant interpretation:** Commit completed work in coherent tested slices going forward, and continue with the next bounded implementation task.

**Inferred user intent:** Avoid accumulating too many unrelated changes before committing.

**Commit (code):** pending for this step.

### What I did
- Added `--bundle` and `--manifest` CLI flags.
- Added JSON/YAML manifest loading for the CLI.
- Kept the built-in Counter demo as the default when both paths are omitted.
- Added `--cpu-timeout`, `--idle-timeout`, `--alarm-interval`, `--idle-interval`, and `--dev-errors` flags.
- Added `cmd/go-go-objects/main_test.go` for input loading behavior.
- Updated `README.md` with custom bundle/manifest usage.
- Ran `gofmt` and `go test ./... -count=1`.

### Why
- A hard-coded demo proves the runtime, but a useful host must accept user-provided object classes and namespace manifests.
- This is a coherent commit-sized slice: CLI configurability plus tests and docs.

### What worked
- The default no-path mode still uses the built-in Counter demo.
- The CLI reads external bundles and YAML manifests.
- `go test ./... -count=1` passes.

### What didn't work
- No major failures in this step.

### What I learned
- The CLI and xgoja provider now have parallel bundle/manifest loading behavior. A future refactor should move file loading into a shared package function to avoid drift.

### What was tricky to build
- The CLI should reject partially configured input. Supplying only `--bundle` or only `--manifest` is ambiguous, so `loadInputs` returns a clear error when the flags are not provided together.

### What warrants a second pair of eyes
- The CLI accepts filesystem paths only. Embedded assets are still provider-specific future work.
- YAML parsing uses `gopkg.in/yaml.v3`, which is already present via the provider dependency.

### What should be done in the future
- Share manifest loading between CLI and xgoja provider.
- Add an example `examples/counter/objects.js` and `examples/counter/durableobjects.yaml` instead of relying only on the embedded string.

### Code review instructions
- Review `cmd/go-go-objects/main.go` for flag semantics and lifecycle behavior.
- Review `cmd/go-go-objects/main_test.go` for accepted and rejected input combinations.
- Validate with `go test ./... -count=1`.
