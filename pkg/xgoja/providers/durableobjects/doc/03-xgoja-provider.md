---
Title: "xgoja Durable Objects provider"
Slug: "go-go-objects-xgoja-provider"
Short: "Embed Durable Objects into xgoja/v2 generated binaries and HTTP servers."
Topics:
- xgoja
- durable-objects
- provider
- embedded-assets
- http
- express
Commands:
- xgoja build
- durableobjects serve
Flags:
- durableobjects-storage-root
- durableobjects-bundle-path
- durableobjects-bundle-asset
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The `go-go-objects-durableobjects` xgoja provider lets generated xgoja/v2 binaries load a Durable Objects bundle, expose a `require("durableobjects")` module, and optionally add a provider-owned `durableobjects serve` command.

Use this path when your application is already an xgoja host or when you want a self-contained generated binary with embedded JavaScript assets.

## Provider registration

A buildspec imports the provider package and selects its runtime module.

```yaml
schema: xgoja/v2
name: durableobjects-counter

providers:
  - id: go-go-objects-durableobjects
    import: github.com/go-go-golems/go-go-objects/pkg/xgoja/providers/durableobjects
    register: Register

runtime:
  modules:
    - provider: go-go-objects-durableobjects
      name: durableobjects
      as: durableobjects
      config:
        storageRoot: ./var/durable-objects
        bundleAsset: counter-bundle
        bundleAssetPath: objects.js
```

Embed the bundle as an asset source:

```yaml
sources:
  - id: counter-bundle
    kind: assets
    from:
      dir: .
    include: [objects.js]

artifacts:
  - id: counter-assets
    type: embedded-assets
    sources: [counter-bundle]
```

## JavaScript module API

Inside xgoja JavaScript, require the module by its selected alias.

```javascript
const durableobjects = require("durableobjects");
```

The module exposes:

| Function | Behavior |
| --- | --- |
| `rpc(namespace, name, method, args)` | Dispatch an RPC method and return the settled result. |
| `fetch(namespace, name, request)` | Dispatch a fetch request and return a response object. |
| `gateway()` / `handler()` | Return a mountable HTTP handler object for the xgoja HTTP provider. |

The module call blocks until the object handler's returned Promise settles or the Durable Objects dispatch timeout expires.

## HTTP composition

Combine the Durable Objects provider with the xgoja HTTP provider when JavaScript should decide where the gateway is mounted.

```javascript
const express = require("express");
const durableobjects = require("durableobjects");

const app = express();
app.use("/rpc", durableobjects.gateway());
app.use("/fetch", durableobjects.gateway());
```

The HTTP provider owns the listener. The Durable Objects provider owns the object manager and returns a mountable handler.

## Direct serve command

The provider can also contribute a direct server command.

```yaml
commands:
  - id: durableobjects-serve
    type: provider.command-set
    provider: go-go-objects-durableobjects
    name: serve
    mount: durableobjects
    config:
      storageRoot: ./var/durable-objects
      bundleAsset: counter-bundle
      bundleAssetPath: objects.js
```

Run it from the generated binary:

```bash
./durableobjects-counter durableobjects serve --addr 127.0.0.1:8787
```

Use `--durableobjects-enabled` when you want command-line Durable Objects section flags to override the provider command config.

```bash
./durableobjects-counter durableobjects serve \
  --durableobjects-enabled \
  --durableobjects-storage-root /tmp/do-storage
```

## Template-generated hosts

Custom xgoja templates can decode the embedded `app.RuntimePlan`, register providers, and construct an `app.Host`. This is useful when a Go application owns the outer `http.Server` and wants to mount Durable Objects into an existing mux.

The Durable Objects provider uses xgoja host services to resolve embedded assets and shared HTTP hosts. Template-generated code should use `app.NewHost` or `app.NewHostWithOptions` instead of bypassing xgoja provider setup.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `missing bundleAsset` | The module config references an asset source that was not embedded. | Add an `embedded-assets` artifact and matching source ID. |
| `/healthz` disappears when mounting | The Durable Objects handler was mounted at `/`. | Mount under `/rpc` and `/fetch` instead. |
| Direct command ignores `--durableobjects-storage-root` | Public Durable Objects section overrides are disabled. | Pass `--durableobjects-enabled` or set the value in command config. |
| Generated binary cannot resolve provider | The buildspec uses the wrong provider ID. | Use the canonical provider ID from the `providers` list. |

## See also

- `go-go-objects-js-api`
- `go-go-objects-overview`
