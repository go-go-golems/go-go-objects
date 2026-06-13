# go-go-objects release readiness notes

## Candidate scope

This release candidate provides an MVP Durable Objects runtime for goja/go-go-goja:

- one live goja runtime per active object identity,
- SQLite-backed object-local storage,
- RPC, fetch, and alarm dispatch,
- HTTP gateway routes under `/rpc` and `/fetch`,
- CLI demo/custom bundle server,
- xgoja provider module with filesystem and embedded asset bundle loading,
- xgoja `durableobjects serve` command provider for generated binaries,
- embeddable HTTP server helper and custom xgoja template example for existing `http.Server` applications,
- automatic `exports.objects` namespace derivation using `CamelCase` to `CAMEL_CASE` conversion.

## Validation commands

Run before tagging:

```bash
go test ./... -count=1
go test ./pkg/durableobjects ./pkg/xgoja/providers/durableobjects \
  -run 'TestConcurrentFirstDispatchStartsOneActor|TestGlazedConfigMapsIntoModuleRPC|TestGeneratedStyleRuntimeLoadsEmbeddedBundleAsset|TestModuleMountsGatewayOnExternalHTTPHost' \
  -race -count=1
docmgr doctor --ticket GOJA-DO-001 --stale-after 30
git status --short
```

## Known limitations

- JavaScript bundles are trusted code, not hostile-code sandboxes.
- Storage quotas are not enforced by the runtime.
- SQLite schema versioning is present through `PRAGMA user_version`, but no multi-version migration chain exists yet.
- The gateway path model treats object names as one URL path segment.
- Automatic HTTP mounting is supported when the Durable Objects provider can see a shared xgoja HTTP host service; embedders can still mount `GatewayService.Handler` manually.
- The generated `durableobjects serve` command is provider-owned; generated binaries must include a `commandProviders` entry for `go-go-objects-durableobjects.serve`.

## Tagging guidance

Use a pre-1.0 tag until the storage migration and quota policies have seen production use, for example:

```bash
git tag v0.1.0-rc1
git push origin v0.1.0-rc1
```
