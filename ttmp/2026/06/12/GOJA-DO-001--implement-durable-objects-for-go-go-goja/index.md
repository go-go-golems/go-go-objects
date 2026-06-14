---
Title: Implement Durable Objects for go-go-goja
Ticket: GOJA-DO-001
Status: active
Topics:
    - goja
    - architecture
    - durable-objects
    - actor-runtime
    - storage
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/modules/database/database.go
      Note: SQLite database module with transactions - storage layer foundation for per-object durable storage
    - Path: go-go-goja/modules/events/events.go
      Note: Go-native EventEmitter - can be used for durable object lifecycle and inter-object communication events
    - Path: go-go-goja/modules/express/express.go
      Note: Express module for HTTP route registration - pattern for exposing durable object RPC/fetch dispatch as a JS module
    - Path: go-go-goja/pkg/engine/factory.go
      Note: RuntimeFactory and RuntimeFactoryBuilder - pattern for composing and creating owned runtimes
    - Path: go-go-goja/pkg/engine/runtime.go
      Note: Core goja runtime lifecycle (VM
    - Path: go-go-goja/pkg/gojahttp/host.go
      Note: gojahttp.Host HTTP dispatch into JS handlers - the pattern durable objects gateway will extend with namespace/name routing
    - Path: go-go-goja/pkg/runtimebridge/runtimebridge.go
      Note: RuntimeOwner interface and context stack - the async-safe scheduling bridge that each actor will need for cross-goroutine dispatch
    - Path: go-go-goja/pkg/runtimeowner/runner.go
      Note: RuntimeOwner with scheduler-based owner-thread semantics - the single-goroutine ownership pattern that durable object actors will build on
    - Path: go-go-goja/pkg/xgoja/providers/http/http.go
      Note: HTTP provider with external host support - the service injection pattern that a durable objects provider could follow
ExternalSources: []
Summary: ""
LastUpdated: 2026-06-12T15:49:10.986018394-04:00
WhatFor: ""
WhenToUse: ""
---










# Implement Durable Objects for go-go-goja

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- goja
- architecture
- durable-objects
- actor-runtime
- storage

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
