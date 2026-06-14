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

