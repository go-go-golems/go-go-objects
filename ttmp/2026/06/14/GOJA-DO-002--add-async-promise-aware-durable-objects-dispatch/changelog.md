# Changelog

## 2026-06-14

- Initial workspace created


## 2026-06-14

Created async dispatch ticket, imported Cloudflare source material, collected local Promise/dispatch evidence, wrote the implementation guide, and added implementation tasks

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/ttmp/2026/06/14/GOJA-DO-002--add-async-promise-aware-durable-objects-dispatch/design-doc/01-async-durable-objects-dispatch-design-guide.md — Primary design guide
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/ttmp/2026/06/14/GOJA-DO-002--add-async-promise-aware-durable-objects-dispatch/tasks.md — Implementation task list


## 2026-06-14

Implemented Promise-aware actor dispatch: added async RPC/fetch/alarm/timeout/transaction tests, split actor invocation from result conversion, awaited returned Promises on the owner thread, and preserved typed errors from Go-backed JS calls

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/durableobjects/actor.go — Promise-aware dispatch implementation
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/durableobjects/durableobjects_test.go — Async dispatch regression tests


## 2026-06-14

Updated public documentation and examples for Promise-aware dispatch: README, release notes, xgoja TypeScript descriptions, and the counter example now describe async RPC/fetch/alarm support and non-goals

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/README.md — Async dispatch user documentation
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/docs/release-notes.md — Async support release notes and validation target
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/examples/counter/objects.js — Async counter example
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/xgoja/providers/durableobjects/durableobjects.go — TypeScript/config description updates


## 2026-06-14

Completed async dispatch validation: normalized owner-call deadline errors to CodeTimeout, passed go test, golangci-lint, gosec, govulncheck, docmgr doctor, and xgoja generated-binary HTTP/direct serve smoke tests

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/durableobjects/actor.go — Timeout normalization for owner-call deadline errors
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/ttmp/2026/06/14/GOJA-DO-002--add-async-promise-aware-durable-objects-dispatch/tasks.md — Validation task completion


## 2026-06-14

Uploaded GOJA-DO-002 bundle to reMarkable and fixed Mermaid sequence diagram actor aliases so the PDF renders without warnings

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/ttmp/2026/06/14/GOJA-DO-002--add-async-promise-aware-durable-objects-dispatch/design-doc/01-async-durable-objects-dispatch-design-guide.md — Mermaid diagram alias cleanup for reMarkable rendering

