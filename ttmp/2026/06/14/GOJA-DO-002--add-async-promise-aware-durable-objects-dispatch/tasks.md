# Tasks

## TODO

- [x] Add failing tests for async RPC, async fetch, async alarm, Promise rejection, pending Promise timeout, and sync-only transaction callbacks.
- [x] Refactor actor dispatch to separate JavaScript invocation, Promise awaiting, and result conversion.
- [x] Implement owner-thread Promise waiting with context/timeout handling and rejection conversion.
- [x] Update README/release notes/examples/TypeScript declarations for Promise-aware async dispatch semantics and non-goals.
- [ ] Run full validation including tests, lint, gosec, govulncheck, docmgr doctor, and xgoja generated binary smoke if examples change.
