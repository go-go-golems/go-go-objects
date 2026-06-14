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


## 2026-06-14

Adapted durableobjects xgoja integration tests and embeddable template from removed RuntimeSpec types to the finalized xgoja v2 RuntimePlan/SourcePlan API

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/examples/templates/durableobjects_http_runtime.go.tmpl — RuntimePlan-based generated template
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/xgoja/providers/durableobjects/durableobjects_test.go — RuntimePlan-based provider test fixtures


## 2026-06-14

Confirmed go-go-goja v2 host-services docs/example commit 63415b9 is local-only for module resolution; deferred go-go-goja go.mod bump until the commit is pushed or tagged

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/go.mod — Dependency remains on published go-go-goja until v2 cutover commit is reachable


## 2026-06-14

Bumped go-go-goja dependency to v0.9.5, which contains the finalized xgoja v2 RuntimePlan API; standalone GOWORK=off tests now pass

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/go.mod — go-go-goja v0.9.5 dependency bump
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/go.sum — Checksums for go-go-goja v0.9.5 transitive dependencies


## 2026-06-14

Fixed async dispatch serialization review finding by adding a context-aware per-actor dispatch gate around invoke, Promise settlement, and result conversion

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/durableobjects/actor.go — Per-actor async dispatch gate
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/durableobjects/durableobjects_test.go — Regression for concurrent async lost updates
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/pkg/durableobjects/manager.go — Initializes actor dispatch gate


## 2026-06-14

Fixed release scaffolding placeholders by replacing XXX with go-go-objects in GoReleaser and release workflow config; snapshot single-target release now succeeds

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/.github/workflows/release.yaml — Disabled docs publish job placeholders
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/.goreleaser.yaml — GoReleaser project/build/package metadata
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/go.sum — go mod tidy removed stale go-go-goja v0.9.1 checksums


## 2026-06-14

Added proper Glazed CLI wiring and embedded help entries for go-go-objects overview, JavaScript API, and xgoja provider usage; help export and snapshot release now work with the Glazed root

### Related Files

- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/cmd/go-go-objects/doc/02-javascript-api.md — Durable Objects JavaScript API help page
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/cmd/go-go-objects/doc/03-xgoja-provider.md — xgoja provider help page
- /home/manuel/workspaces/2026-06-12/goja-durable-objects/go-go-objects/cmd/go-go-objects/main.go — Glazed root command

