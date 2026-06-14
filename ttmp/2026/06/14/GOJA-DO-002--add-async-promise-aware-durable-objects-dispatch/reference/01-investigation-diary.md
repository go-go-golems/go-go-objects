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
RelatedFiles: []
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
