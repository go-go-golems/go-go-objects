---
Title: "go-go-objects docs verb"
Slug: "go-go-objects-docs-verb"
Short: "List, show, and serve go-go-objects documentation from a generated xgoja binary."
Topics:
- durable-objects
- xgoja
- documentation
Commands:
- durableobjects docs
IsTopLevel: false
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The `docs` command set is the durableobjects provider's documentation verb. It exposes the same Glazed help pages bundled by the provider's `HelpSource` through mounted CLI commands, so a generated xgoja binary can list, inspect, and serve the go-go-objects documentation without a separate help-only binary.

Select the verb in a buildspec's `commands` section:

```yaml
commands:
  - id: go-go-objects-docs
    type: provider.command-set
    provider: go-go-objects-durableobjects
    name: docs
    mount: durableobjects
```

The verb builds its own help system from the same embedded filesystem that the `HelpSource` ships, so it works even in binaries that do not select the global help source. The two surfaces read identical bytes and cannot diverge within one binary.

## Commands

### `durableobjects docs list`

List every embedded help page as a Glaze table. Standard Glazed output flags apply, so the list can be rendered as text, JSON, CSV, and so on.

```bash
./my-binary durableobjects docs list --output json
```

Each row carries `slug`, `title`, `sectionType`, `short`, and `topics`.

### `durableobjects docs show <slug>`

Print the rendered Markdown body of one help page, looked up by its Glazed slug.

```bash
./my-binary durableobjects docs show go-go-objects-js-api
```

Use `durableobjects docs list` to discover available slugs. An unknown slug returns an error.

### `durableobjects docs serve --addr <addr>`

Start an HTTP server that exposes the help pages as JSON.

```bash
./my-binary durableobjects docs serve --addr 127.0.0.1:8788
```

Endpoints:

| Method and path | Result |
| --- | --- |
| `GET /docs` | Array of `{slug, title, sectionType, summary}`. |
| `GET /docs/{slug}` | Full page object `{slug, title, sectionType, summary, body, topics, commands, flags}`. |

Unknown slugs return `404`. The server binds to `127.0.0.1:8788` by default.

## Relationship to the help source

The verb is independent of the buildspec `kind: help` source, but selecting both gives the richest experience. With the help source selected, the same pages are available through `help <slug>` (cobra) in addition to the `docs` verb:

```yaml
sources:
  - id: go-go-objects-help
    kind: help
    from:
      provider:
        provider: go-go-objects-durableobjects
        source: go-go-objects
commands:
  - id: go-go-objects-docs
    type: provider.command-set
    provider: go-go-objects-durableobjects
    name: docs
    mount: durableobjects
```

Both the `docs` verb and `help <slug>` read from the same embedded pages and cannot diverge within one binary.

> Note: the JavaScript `require("docs")` docaccess module is available in `goja-repl` but is not yet wired into generated xgoja eval/run/repl runtimes. Exposing provider help sources to JavaScript inside xgoja is tracked as future work and would require wiring the docaccess runtime registrar into the xgoja runtime factory.

## Troubleshooting

| Problem | Cause | Solution |
| --- | --- | --- |
| `docs` command is missing | The buildspec did not select the command set. | Add a `provider.command-set` command entry with `name: docs`. |
| `serve` returns empty `/docs` | The embedded filesystem failed to load. | Rebuild the binary; this should never happen for a clean build. |
| `show <slug>` errors with "no help page" | The slug is wrong. | Run `durableobjects docs list` for valid slugs. |
| Pages differ from `help <slug>` | Impossible in one binary; both read the same embedded bytes. | If observed, the binary was rebuilt between calls. |

## See also

- `go-go-objects-overview`
- `go-go-objects-js-api`
- `go-go-objects-xgoja-provider`
