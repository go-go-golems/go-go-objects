---
Title: Investigation diary
Ticket: GOJA-DO-002
Status: active
Topics:
    - goja
    - durable-objects
    - actor-runtime
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/durableobjects/actor.go
      Note: Step 2 Promise-aware dispatch implementation
    - Path: pkg/durableobjects/durableobjects_test.go
      Note: Step 2 async dispatch tests
ExternalSources: []
Summary: Chronological diary for adding Promise-aware async Durable Objects dispatch.
LastUpdated: 2026-06-14T18:00:00Z
WhatFor: Use this to understand how GOJA-DO-002 was researched and implemented.
WhenToUse: Read before resuming or reviewing async dispatch work.
---


# Diary

## Goal

This diary records the research, design, and implementation work for GOJA-DO-002: adding Promise-aware async dispatch to the `go-go-objects` Durable Objects runtime.

## Step 1: Create the async dispatch ticket and design guide

This step turned the async Durable Objects question into a focused implementation ticket. I researched Cloudflare's async Durable Objects behavior, inspected the current `go-go-objects` actor dispatch code, found existing Promise-waiting patterns in `go-go-goja`, and wrote an intern-facing implementation guide that separates the immediate Promise-aware dispatch milestone from larger Cloudflare compatibility features such as input gates and output gates.

The design intentionally chooses a conservative first implementation: await returned Promises for RPC, fetch, and alarm handlers while keeping one active dispatch serialized per object until that Promise settles. This gives JavaScript authors useful `async` handler support without pretending to implement Cloudflare's full interleaving and gate semantics.

### Prompt Context

**User prompt (verbatim):** "Create a new ticket and Create  a detailed analysis / design / implementation guide that is for a new intern, explaining all the parts of the system needed to understand what it is, with prose paragraphs and bullet point sand pseudocode and diagrams and api references and file references. It should be very clear and technical. Store in the ticket and the nupload to remarkable."

**Assistant interpretation:** Create a new docmgr ticket for async/Promise-aware Durable Objects dispatch, research the current runtime and Cloudflare semantics, write a clear implementation guide, and prepare it for reMarkable delivery.

**Inferred user intent:** Plan the next compatibility milestone before implementing it so the work is evidence-backed and reviewable.

**Commit (code):** pending for this step.

### What I did
- Created ticket `GOJA-DO-002` with title `Add async Promise-aware Durable Objects dispatch`.
- Added a design document and this investigation diary.
- Extracted Cloudflare Durable Objects docs into `sources/` using Defuddle.
- Collected local line evidence into `various/01-line-evidence.md`.
- Wrote `design-doc/01-async-durable-objects-dispatch-design-guide.md`.
- Added concrete implementation tasks covering tests, actor refactor, Promise waiting, docs/examples, and validation.

### Why
- Current actor dispatch calls JavaScript methods synchronously and does not await returned `goja.Promise` values.
- Cloudflare Durable Objects commonly use `async fetch`, async RPC methods, and async `alarm` handlers.
- The implementation needs a clear boundary between Promise-aware handler completion and larger Cloudflare features such as input/output gates.

### What worked
- Cloudflare docs provided direct evidence for async handlers, input gates, output gates, `blockConcurrencyWhile`, and async alarms.
- Existing `go-go-goja` REPL/session code provided concrete local Promise-waiting patterns.
- The current `go-go-objects` actor code has a narrow dispatch seam that can be refactored without redesigning the manager.

### What didn't work
- N/A for this step; this was research and ticket setup.

### What I learned
- Promise-aware dispatch is a prerequisite for Cloudflare-like Durable Objects, but not the same thing as Cloudflare's full concurrency model.
- The first implementation should keep actor dispatch serialized while awaiting returned Promises to avoid introducing ungated interleaving races.

### What was tricky to build
- The main design trap was conflating three different concerns: awaiting a returned handler Promise, allowing event interleaving during awaits, and implementing output gates for storage writes. The guide separates these into phases and only puts Promise-aware completion in the first implementation scope.

### What warrants a second pair of eyes
- Whether `CPUTimeout` should remain the user-facing timeout name once it also covers pending Promise wait time.
- Whether Promise waiting should be factored into `go-go-goja` after the local Durable Objects implementation proves the exact helper API needed.

### What should be done in the future
- Implement the ticket tasks in order: tests first, actor refactor, Promise waiting, docs/examples, validation.
- Upload the final design bundle to reMarkable after implementation docs are complete.

### Code review instructions
- Start with `design-doc/01-async-durable-objects-dispatch-design-guide.md`.
- Then inspect `pkg/durableobjects/actor.go` and `go-go-goja/pkg/replsession/evaluate.go` for the current and proposed Promise handling seams.
- Validate the ticket with `docmgr doctor --ticket GOJA-DO-002 --stale-after 30`.

### Technical details
- Ticket path: `ttmp/2026/06/14/GOJA-DO-002--add-async-promise-aware-durable-objects-dispatch`.
- Primary design doc: `design-doc/01-async-durable-objects-dispatch-design-guide.md`.
- Evidence file: `various/01-line-evidence.md`.

## Step 2: Add tests and implement Promise-aware dispatch

This step implemented the first async dispatch milestone. I added regression tests that intentionally failed against the synchronous actor implementation, then refactored `Actor.Dispatch` so RPC, fetch, and alarm handlers can return either plain values or Promises. The actor now waits for a returned Promise to settle before converting the result into a Durable Objects `Result`.

The implementation keeps the conservative runtime model from the design guide: one dispatch remains active until its Promise settles. That means async handlers now work, but the runtime still does not implement Cloudflare-style event interleaving, input gates, or output gates.

### Prompt Context

**User prompt (verbatim):** "Then add tasks, and implement it step by step, keeping a detailed diary as you work, committing at appropriate intervals."

**Assistant interpretation:** Convert the design ticket into tasks and begin implementation in focused, documented, committed slices.

**Inferred user intent:** Move from design to working code while preserving a clear implementation history and review trail.

**Commit (code):** pending for this step.

### What I did
- Added implementation tasks to `GOJA-DO-002`.
- Extended the test counter object with async RPC, rejected RPC, pending Promise, async fetch, async alarm, and bad async transaction cases.
- Added tests for:
  - fulfilled async RPC;
  - rejected async RPC;
  - pending Promise timeout;
  - async transaction callback rejection;
  - fulfilled async fetch;
  - existing alarm dispatch through an async `alarm()` handler.
- Refactored `pkg/durableobjects/actor.go` into invocation, Promise awaiting, and conversion phases.
- Added owner-thread Promise polling using `RuntimeOwner.Call()`.
- Preserved typed `*durableobjects.Error` values that are thrown through Go-backed JS callbacks.
- Kept `state.storage.transaction` synchronous-only.

### Why
- The current implementation immediately converted returned `goja.Value` values and therefore returned Promise objects instead of fulfilled values.
- Cloudflare Durable Objects code commonly uses async RPC/fetch/alarm handlers, so this was the most important semantic gap after the MVP runtime landed.

### What worked
- The initial targeted test run failed exactly on the expected sync-only gaps:
  - `TestAsyncRPCDispatchAwaitsFulfilledPromise` returned a non-number Promise result.
  - `TestAsyncRPCDispatchPropagatesRejectedPromise` succeeded instead of surfacing rejection.
  - `TestAsyncRPCDispatchPendingPromiseTimesOut` succeeded instead of timing out.
  - `TestAsyncFetchDispatchAwaitsFulfilledPromise` returned a default 200/empty body instead of awaited 202/4.
- After the refactor, the targeted async/alarm tests passed.
- `GOWORK=off go test ./... -count=1` passed.

### What didn't work
- The first refactor draft tried to format rejection values with an invalid `goja.Value` runtime access pattern and referenced non-existent error helper APIs. I corrected this before writing the real file by inspecting `pkg/durableobjects/errors.go`.
- Go-backed JS errors thrown through `panic(vm.NewGoError(err))` arrive wrapped by goja. The actor now attempts to recover nested durable errors from `*goja.Exception` so `CodeBadRequest` from async transaction rejection is preserved.

### What I learned
- Promise awaiting fits cleanly between invocation and conversion. Trying to keep conversion inside the original owner call would make Promise waiting harder to reason about.
- The existing active-dispatch counter did not need changes: keeping the dispatch active while awaiting the Promise also protects the actor from idle eviction.

### What was tricky to build
- The main sharp edge was error-code preservation. A storage transaction callback error crosses the JS boundary as a Go error wrapped in a goja exception. If actor dispatch blindly wraps it as `CodeExecutionError`, callers lose the more precise `CodeBadRequest`. The fix was to add a small `durableErrorFrom` extraction helper and use it before wrapping errors.
- The Promise wait loop must inspect `promise.State()` and `promise.Result()` on the runtime owner thread. The implementation polls through short `RuntimeOwner.Call()` calls and sleeps outside the owner.

### What warrants a second pair of eyes
- The timeout path currently uses the existing `CPUTimeout` as the full dispatch/Promise wait budget. This matches the design, but the option name may become confusing.
- Rejection formatting currently prefers `error.message` when available and otherwise falls back to `value.String()`. Rich stack traces are still a future improvement.

### What should be done in the future
- Update README, release notes, examples, and TypeScript declarations to document Promise-aware dispatch and non-goals.
- Run the full lint/security validation suite after documentation updates.

### Code review instructions
- Start with `pkg/durableobjects/actor.go` and read the flow in this order: `Dispatch`, `invokeDispatch`, `awaitValue`, `convertDispatchValue`.
- Then review `pkg/durableobjects/durableobjects_test.go` for the async behavior contract.
- Validate with `GOWORK=off go test ./... -count=1`.

### Technical details
- Focused failing command before implementation:
  `GOWORK=off go test ./pkg/durableobjects -run 'TestAsync|TestAlarmDispatchWakesEvictedActor|TestDispatchDueAlarmsReconcilesMissingIndex|TestAlarmSchedulerTick' -count=1`
- Passing command after implementation:
  `GOWORK=off go test ./... -count=1`
