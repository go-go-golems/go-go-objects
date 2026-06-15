---
Title: 'Design: Expose go-go-objects docs to xgoja'
Ticket: GOJA-DO-003
Status: active
Topics:
    - goja
    - xgoja
    - durable-objects
    - documentation
    - architecture
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: go-go-goja/examples/xgoja/09-provider-shipped-help-docs/xgoja.yaml
      Note: Reference buildspec for a provider-shipped kind:help source
    - Path: go-go-goja/pkg/docaccess/glazed/provider.go
      Note: HelpSystem->docaccess adapter; same GetSectionWithSlug lookup the docs verb reuses
    - Path: go-go-goja/pkg/xgoja/app/framework.go
      Note: loadConfiguredHelpSources resolves a provider HelpSource and loads it into the root HelpSystem
    - Path: go-go-goja/pkg/xgoja/providerapi/commands.go
      Note: CommandSetProvider/CommandSetContext shapes used by the docs verb
    - Path: go-go-goja/pkg/xgoja/providerapi/help.go
      Note: The providerapi.HelpSource hook (Name/FS/Root) that bundles Glazed docs into xgoja binaries
    - Path: go-go-objects/pkg/xgoja/providers/durableobjects/durableobjects.go
      Note: Register() is where the HelpSource + docs CommandSetProvider entries are added
ExternalSources: []
Summary: Intern-facing design and implementation guide for bundling go-go-objects Glazed help pages into generated xgoja binaries (HelpSource) and exposing them through a provider-owned docs verb (CommandSetProvider).
LastUpdated: 2026-06-15T22:00:00-04:00
WhatFor: ""
WhenToUse: ""
---







# Expose go-go-objects documentation to xgoja

This document is written for an engineer who is new to the `go-go-golems` JavaScript-on-Go stack. It explains, from first principles, how documentation flows from a Markdown file to a JavaScript-visible API inside a generated `xgoja` binary, and then gives a precise, phased implementation plan for doing two things:

1. **Bundle** the `go-go-objects` Glazed help pages into any generated xgoja binary that selects them.
2. **Expose** those same pages through a provider-owned "verb" (a mounted CLI command set) so a generated binary can list, show, and serve the documentation.

Read sections 1–4 to understand the system; read sections 5–9 to implement the change.

---

## 1. Executive summary

`go-go-objects` already ships three high-quality Glazed help pages (overview, JavaScript API, xgoja provider). Today those pages are embedded only by the standalone `go-go-objects` binary and are invisible to generated xgoja binaries that use the Durable Objects provider.

`go-go-goja` already provides the exact mechanism we need: an xgoja provider can declare a `HelpSource` (a packaged `fs.FS` of Glazed Markdown) and a `CommandSetProvider` (a factory for mounted CLI commands). The generated binary's app layer turns a selected `HelpSource` into a loaded `help.HelpSystem`, and turns a selected `CommandSetProvider` into real cobra subcommands.

The change is small and localized to the `go-go-objects` Durable Objects provider package:

- Move the three help Markdown files from `cmd/go-go-objects/doc/` into the provider package and `go:embed` them.
- Add one `providerapi.HelpSource{...}` entry to the existing `Register` call so the docs become a selectable help source.
- Add one `providerapi.CommandSetProvider{...}` entry (the `docs` verb) whose `NewCommandSet` returns `list` / `show` / `serve` commands that read the embedded help.
- Wire the new source and command into a buildspec's `sources` and `commands` sections so a generated binary actually exposes them.

No changes to `go-go-goja` are required. The mechanism is already production code, exercised by the `loupedeck` provider in example `09-provider-shipped-help-docs` and by the `fixture` command provider in example `05-command-provider`.

---

## 2. Problem statement and scope

### Problem

The Durable Objects xgoja provider (`go-go-objects/pkg/xgoja/providers/durableobjects`) lets a generated binary `require("durableobjects")` and call `rpc`/`fetch`/`gateway`. But the documentation that explains how to write Durable Objects, how the storage API works, and how to configure the provider is trapped inside the standalone `cmd/go-go-objects` binary. A user who builds a custom xgoja binary that embeds Durable Objects has no in-binary way to read that documentation through `help`, through the `docs` JavaScript module, or through a dedicated command.

### In scope

- Relocating the existing help Markdown into the provider package so it can be embedded by `go:embed`.
- Registering a `providerapi.HelpSource` for the Durable Objects provider.
- Adding a `docs` `providerapi.CommandSetProvider` (the "verb") with `list`, `show`, and `serve` subcommands.
- Documenting the buildspec stanzas (`kind: help` source and `provider.command-set` command) that select them.
- Providing an intern-grade explanation of the docaccess and xgoja command-provider subsystems.

### Out of scope

- Writing new help pages. The three existing pages are reused as-is.
- Changing the Durable Objects runtime, storage, or gateway behavior.
- Modifying `go-go-goja` core. Everything needed already exists there.
- Backwards-compatibility shims for the old `cmd/go-go-objects/doc` import path (see Decision D3; the standalone binary's fate is a maintainer decision).

### Success criteria

1. A generated xgoja binary that selects the Durable Objects help source exposes the pages via `help <slug>` (cobra). (The JavaScript `require("docs")` docaccess module is a `goja-repl` feature and is **not** wired into generated xgoja runtimes today; see Open Questions.)
2. The same binary, when given the `docs` command in its buildspec, exposes `durableobjects docs list`, `durableobjects docs show <slug>`, and `durableobjects docs serve`.
3. No edits to `go-go-goja` are required to achieve 1 and 2.

---

## 3. Glossary

- **Glazed**: The CLI framework used across `go-go-golems`. Provides `cmds.Command`, `schema.Section`, `fields`, and the `help.HelpSystem`.
- **Glazed help page**: A Markdown file whose YAML frontmatter uses Glazed keys (`Title`, `Slug`, `Short`, `Topics`, `SectionType`, ...). Rendered by the `help` cobra command.
- **goja**: A pure-Go ECMAScript 5.1+ interpreter (`github.com/dop251/goja`).
- **go-go-goja**: The `go-go-golems` layer on top of goja: owned runtimes, async event loop, native modules, the REPL, and the xgoja code generator.
- **xgoja**: A code generator (`go-go-goja/cmd/xgoja`) that turns an `xgoja.yaml` "buildspec" into a Go binary with selected provider modules and commands.
- **Provider**: A Go package that calls `registry.Package(id, entries...)` to contribute modules, verb sources, help sources, capabilities, and command sets.
- **HelpSource**: A provider-owned `fs.FS` + `Root` of Glazed Markdown (`providerapi.HelpSource`). Selected in a buildspec via `kind: help`.
- **CommandSetProvider**: A provider-owned factory that returns Glazed commands (`providerapi.CommandSetProvider`). The thing a buildspec turns into a mounted cobra command — i.e. a "verb".
- **Verb**: Informal term for a mounted Glazed command (whether it came from a JS verb source or a Go `CommandSetProvider`).
- **docaccess**: The `go-go-goja/pkg/docaccess` subsystem that gives JavaScript a uniform query API over documentation sources.
- **docs module**: The runtime `require("docs")` module built by the docaccess registrar.

---

## 4. Current-state architecture (evidence-based)

This section explains the moving parts you must understand before implementing. Every claim is anchored to a file.

### 4.1 The big picture: how Markdown becomes a JS-visible doc

The pipeline has five stages. Each stage is a small, testable unit; together they move a Markdown file from disk into a JavaScript object.

```
                          (1) embed                (2) buildspec selects         (3) app loads
   *.md  ──────────────────────────►  provider   ──────────────────────────►  generated binary
 (Glazed frontmatter)        go:embed  HelpSource     xgoja.yaml "kind:help"   help.HelpSystem
      │                                   │                                              │
      │                                   │                                              │  (4) glazed docaccess
      │                                   │                                              │      provider wraps it
      ▼                                   ▼                                              ▼
 help.HelpSystem  ◄──── LoadSectionsFromFS(FS, Root)  ◄──── providerapi.HelpSource{FS,Root}
      │
      │ (5) registrar builds Hub + "docs" native module
      ▼
 require("docs").bySlug(...)        help <slug>  (cobra)
```

Stage (1) is `go:embed` of the Markdown into the provider package. Stage (2) is the user's `xgoja.yaml`. Stages (3)–(5) are owned by `go-go-goja` and already work. GOJA-DO-003 only touches stages (1) and (2) on the go-go-objects side.

The two terminal surfaces are:

- The cobra `help` command (and `help <slug>`), wired by `help_cmd.SetupCobraRootCommand(helpSystem, root)`.
- The JavaScript `require("docs")` module, wired by the docaccess runtime registrar.

A generated binary can expose one, the other, or both, depending on which built-in commands (`eval`, `run`, `repl`) and sources it selects.

### 4.2 The docaccess data model

File: `go-go-goja/pkg/docaccess/model.go`.

docaccess is intentionally storage-agnostic. It defines four small types that every documentation source must speak:

```go
type SourceKind string
// SourceKindGlazedHelp | SourceKindJSDoc | SourceKindPlugin | SourceKindDocmgr

type SourceDescriptor struct {
    ID, Title, Summary string
    Kind               SourceKind
    RuntimeScoped      bool
    Metadata           map[string]any
}

type EntryRef struct { SourceID, Kind, ID string }   // a stable pointer to one doc

type Entry struct {
    Ref       EntryRef
    Title, Summary, Body string
    Topics, Tags         []string
    Path, KindLabel      string
    Related              []EntryRef
    Metadata             map[string]any
}

type Query struct {
    Text                      string
    SourceIDs, Kinds          []string
    Topics, Tags              []string
    Limit                     int
}
```

Mental model: a `SourceDescriptor` advertises a documentation source; an `EntryRef` points at exactly one document within a source; an `Entry` is the fully materialized document. A `Query` filters across many sources.

### 4.3 The Provider interface and the Hub

File: `go-go-goja/pkg/docaccess/provider.go`, `go-go-goja/pkg/docaccess/hub.go`.

A source implements one interface:

```go
type Provider interface {
    Descriptor() SourceDescriptor
    List(ctx) ([]EntryRef, error)
    Get(ctx, EntryRef) (*Entry, error)
    Search(ctx, Query) ([]Entry, error)
}
```

The `Hub` is a thread-safe registry of providers keyed by `SourceDescriptor.ID`:

- `Register(provider)` rejects empty/duplicate IDs.
- `Sources()` returns all descriptors, sorted by ID.
- `Get(ctx, ref)` dispatches to the provider named in `ref.SourceID`.
- `Search(ctx, q)` fans the query out to every provider (or a filtered subset via `q.SourceIDs`), merges, sorts, and applies `q.Limit`.

This fan-out is why the `docs` module can search "across Glazed help, jsdoc, and plugin metadata" in one call.

### 4.4 The Glazed adapter: HelpSystem → docaccess.Provider

File: `go-go-goja/pkg/docaccess/glazed/provider.go`.

This is the adapter that matters for GOJA-DO-003. It wraps a `*help.HelpSystem`:

```go
const EntryKindHelpSection = "help-section"

func NewProvider(sourceID, title, summary string, hs *help.HelpSystem) (*Provider, error)

// Descriptor()  -> {ID: sourceID, Kind: SourceKindGlazedHelp, Title, Summary}
// Get(ref)      -> hs.GetSectionWithSlug(ref.ID)   // ref.Kind must be "help-section"
// Search(q)     -> hs.QuerySections("")            // returns every section
```

Each Glazed `Section` becomes an `Entry` whose `Ref.ID` is the section's `Slug` and whose `Metadata` carries `slug`, `commands`, `flags`, `sectionType`, `isTopLevel`, `showPerDefault`, `order`. That metadata is exactly what makes `docs.bySlug("default-help", "go-go-objects-overview")` work.

### 4.5 The runtime registrar: builds the Hub and the `docs` module

File: `go-go-goja/pkg/docaccess/runtime/registrar.go`.

`Registrar` is an engine module registrar. At runtime-module registration time it:

1. Builds a `docaccess.Hub`.
2. For each `HelpSource` in `Config.HelpSources`, wraps the `*help.HelpSystem` with the glazed adapter (section 4.4) and registers it.
3. For each `JSDocSource`, wraps the jsdoc store similarly.
4. If the host loaded plugins, registers the plugin-manifests provider.
5. Stores the hub under context key `RuntimeHubContextKey = "docaccess.hub"`.
6. Registers a native module named `docs` (configurable via `Config.ModuleName`) whose loader wires the hub to JavaScript.

Important type-name disambiguation — there are **two** structs named `HelpSource`:

| Struct | Lives in | Carries | Used by |
| --- | --- | --- | --- |
| `runtime.HelpSource` | `docaccess/runtime/registrar.go:23` | an already-built `*help.HelpSystem` | hand-built hosts like `goja-repl` |
| `providerapi.HelpSource` | `xgoja/providerapi/help.go:15` | a raw `fs.FS` + `Root` of Markdown | generated xgoja binaries |

The app layer is the bridge between them: it reads the providerapi `HelpSource.FS/Root`, calls `helpSystem.LoadSectionsFromFS(FS, Root)` to build a HelpSystem, and (when a `repl`/`eval` command is present) the registrar turns that HelpSystem into a `docs` module. Section 4.7 covers that bridge.

### 4.6 The `docs` JavaScript API

File: `go-go-goja/pkg/doc/15-docs-module-guide.md` (spec), `docaccess/runtime/registrar.go:141` (loader).

```javascript
const docs = require("docs")

docs.sources()                              // -> SourceDescriptor[]
docs.search({ text, sourceIds, kinds, topics, tags, limit })  // -> Entry[]
docs.get({ sourceId, kind, id })           // -> Entry | null
docs.byID(sourceId, kind, id)              // convenience
docs.bySlug(sourceId, slug)                // convenience for help-section
docs.bySymbol(sourceId, symbol)            // convenience for jsdoc symbols
```

Each `Entry` arrives as a plain object: `{ ref, title, summary, body, topics, tags, path, kindLabel, related, metadata }`. `metadata.commands`, `metadata.flags`, and `metadata.sectionType` are populated for Glazed help sections.

For go-go-objects, after this work a generated binary that selects the help source and includes a `repl`/`eval` command will let JavaScript do:

```javascript
docs.bySlug("go-go-objects", "go-go-objects-js-api").body
```

### 4.7 The xgoja providerapi: HelpSource, CommandSetProvider, Package

File: `go-go-goja/pkg/xgoja/providerapi/{help,commands,module,provider_registry,capabilities}.go`.

A provider contributes capabilities by registering a **Package**. A package is a bag of typed entries:

```go
type Package struct {
    ID                  string
    Modules             map[string]Module             // require()-able native modules
    VerbSources         map[string]VerbSource         // packaged *.js verb files
    HelpSources         map[string]HelpSource         // packaged *.md help pages  <-- we add one
    PackageCapabilities map[string]PackageCapability   // config/init/host hooks
    CommandSetProviders map[string]CommandSetProvider // mounted CLI commands      <-- we add one
}
```

Registration is variadic and order-independent:

```go
registry.Package("my-package",
    providerapi.Module{...},
    providerapi.HelpSource{Name: "docs", FS: docFS, Root: "."},
    providerapi.CommandSetProvider{Name: "docs", DefaultMount: "my", NewCommandSet: ...},
    providerapi.WithPackageCapability(myCap),
)
```

The two entries GOJA-DO-003 adds:

**`HelpSource`** (`providerapi/help.go`):

```go
type HelpSource struct {
    Name        string  // unique within the package; referenced as source.Name in buildspecs
    Description string
    FS          fs.FS   // required; must contain Glazed Markdown
    Root        string  // optional; defaults to "."
}
```

It validates that `Name` is non-empty and `FS` is non-nil. The generated binary resolves it by `ResolveHelpSource(packageID, sourceName)`.

**`CommandSetProvider`** (`providerapi/commands.go`):

```go
type CommandSetProvider struct {
    Name          string                         // unique within the package
    DefaultMount  string                         // cobra parent command if buildspec omits mount
    Description   string
    ConfigSchema  json.RawMessage                // optional JSON schema for command config
    NewCommandSet func(CommandSetContext) (*CommandSet, error)
}

type CommandSet struct {
    Commands     []cmds.Command                  // Glazed Bare/Writer/Glaze commands
    ParserConfig *cli.CobraParserConfig          // optional; else defaults are used
}

type CommandSetContext struct {
    Context         context.Context
    PackageID, Name, Mount string
    Config          json.RawMessage               // from the buildspec command's `config:`
    Host            HostServices                  // exposes AssetResolver()
    Providers       *ProviderRegistry
    RuntimeFactory  RuntimeFactory                // build xgoja runtimes if needed
    SelectedModules []ModuleDescriptor
    Sources         SourceRegistry
}
```

This is the full surface the `docs` verb's `NewCommandSet` will receive.

### 4.8 Runtime wiring: how a generated binary loads help

File: `go-go-goja/pkg/xgoja/app/framework.go`.

Every generated xgoja root command gets the "root framework" installed by `installRootFramework`. That function:

1. Adds the logging section (`logging.AddLoggingSectionToRootCommand`).
2. Creates a fresh `help.NewHelpSystem()`.
3. Loads xgoja's own built-in docs (`xgojadoc.AddDocToHelpSystem`).
4. Calls `loadConfiguredHelpSources(helpSystem, runtimePlan, opts)`.
5. Calls `help_cmd.SetupCobraRootCommand(helpSystem, root)` — this attaches the `help` command and slug lookups.

`loadConfiguredHelpSources` (framework.go:61) is the bridge. It reads the runtime plan's help sources (`runtimePlan.sourcesByKind(SourceKindHelp)`) and, for each, picks one of three origins:

```go
hasProvider := source.ProviderID() != "" || source.Source != ""
switch {
case hasProvider:                                    // provider-shipped
    ps, _ := opts.Providers.ResolveHelpSource(source.ProviderID(), source.Source)
    helpSystem.LoadSectionsFromFS(ps.FS, ps.Root)    // <- our HelpSource lands here
case source.Embed:                                   // copied-into-workspace + embedded
    helpSystem.LoadSectionsFromFS(opts.EmbeddedHelp, source.Path)
default:                                             // runtime filesystem
    helpSystem.LoadSectionsFromFS(os.DirFS(source.Path), ".")
}
```

Two consequences that shape the design:

- **Silent no-op.** If no help source is configured, `loadConfiguredHelpSources` returns `nil` without error. Registering a `HelpSource` in Go is necessary but not sufficient — the buildspec must also select it.
- **Duplicate detection.** Source IDs must be unique within a binary; duplicates return an error.

### 4.9 Runtime wiring: how a generated binary mounts a command provider

File: `go-go-goja/pkg/xgoja/app/command_providers.go`.

`Host.AttachProviderCommands(root)` iterates `runtimePlan.runtimeCommands()` and, for each command whose `Type == "provider.command-set"`, calls `AttachCommandProvider`:

```go
provider, _ := h.Providers.ResolveCommandSetProvider(instance.ProviderID(), instance.Name)
mount := instance.Mount
if mount == "" { mount = provider.DefaultMount }
set, err := h.newCommandSet(instance, provider, mount)   // builds CommandSetContext, calls NewCommandSet
commands := applyMountToCommands(set.Commands, mount)     // prepends mount to each command's Parents
glazedcli.AddCommandsToRootCommand(root, commands, ...)   // mounts as cobra subcommands
```

`h.newCommandSet` builds the `CommandSetContext` from the runtime plan, including `Config` (JSON-marshalled from the buildspec command's `config:` map), `Host`, `Providers`, `RuntimeFactory`, `SelectedModules`, and a scoped `SourceRegistry`.

So the lifecycle of the docs verb is: buildspec `commands:` entry → runtime plan `CommandPlan` → `AttachProviderCommands` → `NewCommandSet(ctx)` → Glazed commands mounted under `mount`.

### 4.10 The buildspec, the runtime plan, and the generator

These three artifacts form the contract between an author and a generated binary.

**Buildspec** (`xgoja.yaml`, `schema: xgoja/v2`). Human-authored. Declares providers, selected runtime modules, sources, commands, and artifacts. The stanzas GOJA-DO-003 cares about:

```yaml
providers:
  - id: go-go-objects-durableobjects
    import: github.com/go-go-golems/go-go-objects/pkg/xgoja/providers/durableobjects
    register: Register

sources:
  - id: go-go-objects-help               # <-- selects the HelpSource
    kind: help
    from:
      provider:
        provider: go-go-objects-durableobjects
        source: go-go-objects            # must match providerapi.HelpSource.Name

commands:
  - id: go-go-objects-docs               # <-- selects the docs verb
    type: provider.command-set
    provider: go-go-objects-durableobjects
    name: docs                           # must match providerapi.CommandSetProvider.Name
    mount: durableobjects
```

**Runtime plan** (embedded JSON, decoded into `app.RuntimePlan`). Machine-generated from the buildspec by `cmd/xgoja`. The relevant slices are:

- `RuntimePlan.Sources []SourcePlan` — each `SourcePlan{ID, Kind, Provider, Source, Path, Embed, ...}`. A provider help source has `Kind == "help"`, non-empty `Provider`/`Source`.
- `RuntimePlan.Commands []CommandPlan` — each `CommandPlan{ID, Type, Name, Mount, Provider, Config, ...}`. A command-set verb has `Type == "provider.command-set"`.

**Generator template** (`cmd/xgoja/internal/generate/templates/main.go.tmpl`). Emits `main.go`:

```go
registry := providerapi.NewProviderRegistry()
must(durableobjectsprovider.Register(registry))   // our package, with HelpSource + docs verb
runtimePlan := decodeRuntimePlan()                // embedded JSON
// ... build host, attach commands, execute root
```

The intern does **not** edit the template. The provider's `Register` function is the only Go code that changes.

### 4.11 The current go-go-objects provider and standalone binary

File: `go-go-objects/pkg/xgoja/providers/durableobjects/durableobjects.go`, `serve.go`; `go-go-objects/cmd/go-go-objects/main.go`.

The provider today registers one package (`go-go-objects-durableobjects`, `PackageID` const at `durableobjects.go:28`) with:

- `providerapi.Module{Name: "durableobjects", ...}` — the JS module exposing `rpc`, `fetch`, `gateway`, `handler`.
- `providerapi.CommandSetProvider{Name: "serve", DefaultMount: "durableobjects", ...}` — the gateway server command.
- `providerapi.WithPackageCapability(capability)` — a capability implementing `GlazedConfigSectionCapability`, `XGojaConfigSectionCapability`, `HostServiceContributionCapability`, and `RuntimeInitializerCapability`.

It registers **no** `HelpSource` and **no** docs command. That is the gap.

The standalone binary (`cmd/go-go-objects/main.go`) is the only place that currently loads the docs:

```go
helpSystem := help.NewHelpSystem()
doc.AddDocToHelpSystem(helpSystem)        // cmd/go-go-objects/doc/doc.go embeds *.md
help_cmd.SetupCobraRootCommand(helpSystem, root)
```

The three Markdown files live at `cmd/go-go-objects/doc/{01-overview,02-javascript-api,03-xgoja-provider}.md`. They are high quality and reuse-ready; GOJA-DO-003 relocates them so both the provider and the standalone binary can share them.

---

## 5. Gap analysis

| Need | Current state | Gap |
| --- | --- | --- |
| Generated binary can show go-go-objects help via `help <slug>` | Not possible; docs live only in the standalone binary. | No `providerapi.HelpSource` is registered. |
| Generated binary can query docs from JS via `require("docs")` | Not wired: the docaccess runtime registrar is added by `goja-repl` but not by the xgoja runtime factory. The HelpSource only feeds the cobra `help` system. | Tracked as a future enhancement (wire `docaccess/runtime.NewRegistrar` into `pkg/xgoja/app/factory.go`). |
| Generated binary has a dedicated `docs` command | Not possible; only `serve` command set exists. | No `docs` `CommandSetProvider`. |
| Docs are authored once, shared by provider + standalone binary | Duplicated knowledge; only the binary embeds them. | Docs are under `cmd/...`, not under the importable provider package. |
| An intern can implement this without touching `go-go-goja` | Mechanism exists and is tested. | No guide ties the pieces together for go-go-objects. |

The root cause is a packaging gap, not a missing feature. Every primitive required (HelpSource, CommandSetProvider, docaccess, the app loader) already exists in `go-go-goja` and is exercised by existing examples.

---

## 6. Proposed architecture

### 6.1 Target file layout

Move the docs into the provider package so the provider and the standalone binary both import them from one source of truth.

```
go-go-objects/
  pkg/xgoja/providers/durableobjects/
    doc/
      doc.go                          # //go:embed *.md  +  AddDocToHelpSystem(help.HelpSystem)
      01-overview.md                  # moved from cmd/go-go-objects/doc/
      02-javascript-api.md            # moved from cmd/go-go-objects/doc/
      03-xgoja-provider.md            # moved from cmd/go-go-objects/doc/
      04-docs-verb.md                 # NEW: documents the docs verb itself (optional, recommended)
    durableobjects.go                 # +HelpSource entry, +docs CommandSetProvider entry
    docs.go                           # NEW: the docs verb (CommandSetProvider + commands)
    serve.go                          # unchanged
  cmd/go-go-objects/
    main.go                           # import the provider's doc package instead of its own
    doc/                              # REMOVED (or reduced to a re-embed shim only if kept)
```

Why a dedicated `doc/` package: `go:embed` only sees files in the same directory tree as the embedding `.go` file. A self-contained `doc` package gives the provider a stable embed root (`"."`) that matches the `Root` field of the `HelpSource` and the convention used by `go-go-goja/pkg/xgoja/doc` and `cmd/go-go-objects/doc/doc.go`.

### 6.2 The doc embed package

`pkg/xgoja/providers/durableobjects/doc/doc.go` mirrors the existing pattern:

```go
package doc

import (
    _ "embed"
    "github.com/go-go-golems/glazed/pkg/help"
)

//go:embed *.md
var docFS embed.FS

// AddDocToHelpSystem loads every Glazed Markdown page in this package.
func AddDocToHelpSystem(hs *help.HelpSystem) error {
    return hs.LoadSectionsFromFS(docFS, ".")
}

// HelpFS returns the embedded filesystem and root used by providerapi.HelpSource.
func HelpFS() (fs.FS, string) {
    sub, err := fs.Sub(docFS, ".")
    if err != nil { return docFS, "." }
    return sub, "."
}
```

Notes for the intern:

- Import `_ "embed"` is the project convention (see AGENT.md). The blank identifier keeps the import for the compiler directive.
- `embed.FS` already satisfies `fs.FS`, but `fs.Sub(docFS, ".")` yields a cleaner subtree so `LoadSectionsFromFS` does not see the leading `./`. Returning a `(fs.FS, string)` pair matches the `HelpSource{FS, Root}` shape exactly and avoids drift.

### 6.3 Register the HelpSource

Edit `durableobjects.go` `Register` to add one entry. The HelpSource `Name` becomes the buildspec `source:` value.

```go
import (
    "github.com/go-go-golems/go-go-objects/pkg/xgoja/providers/durableobjects/doc"
)

func Register(registry *providerapi.ProviderRegistry) error {
    docFS, docRoot := doc.HelpFS()
    capability := newCapability()
    return registry.Package(PackageID,
        providerapi.Module{ /* unchanged */ },
        providerapi.CommandSetProvider{ Name: "serve", /* unchanged */ },

        // (1) bundle docs into any binary that selects them
        providerapi.HelpSource{
            Name:        "go-go-objects",
            Description: "Durable Objects overview, JavaScript API, and xgoja provider guide",
            FS:          docFS,
            Root:        docRoot,
        },

        // (2) expose docs through a mounted command set
        providerapi.CommandSetProvider{
            Name:         "docs",
            DefaultMount: "durableobjects",
            Description:  "List, show, and serve go-go-objects documentation",
            NewCommandSet: newDocsCommandSet,
        },

        providerapi.WithPackageCapability(capability),
    )
}
```

Why `Name: "go-go-objects"` (not `"docs"`): the docaccess source ID ultimately surfaced to JavaScript is the runtime-plan source `ID`, not the provider `HelpSource.Name`. But the `Name` is what a buildspec references under `from.provider.source`. Keeping it human-readable avoids confusion and lets multiple providers each ship a help source without colliding.

### 6.4 The `docs` verb

New file `pkg/xgoja/providers/durableobjects/docs.go`. It owns a `CommandSetProvider` factory and three Glazed commands. The verb reads exclusively from the embedded help system built in-memory; it does not start the Durable Objects gateway.

Design goals for the verb:

- `durableobjects docs list` — Glaze command; one row per help page (`slug`, `title`, `sectionType`, `topics`).
- `durableobjects docs show <slug>` — Writer command; prints the rendered Markdown body.
- `durableobjects docs serve` — Bare command; starts an HTTP server that exposes the help system (mirrors the standalone binary's doc-serving role, generalized).

All three build their own `*help.HelpSystem` from the embedded FS so the verb works even when the binary does not select the help source globally. (A binary that *also* selects the help source will have the same docs available via `help <slug>`. The JavaScript `require("docs")` surface is a goja-repl feature not yet wired into xgoja runtimes.)

#### 6.4.1 Command set factory

```go
package durableobjectsprovider

import (
    "github.com/go-go-golems/glazed/pkg/cmds"
    "github.com/go-go-golems/glazed/pkg/help"
    "github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
    "github.com/go-go-golems/go-go-objects/pkg/xgoja/providers/durableobjects/doc"
)

const docsCommandSetName = "docs"

func newDocsCommandSet(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
    hs := help.NewHelpSystem()
    if err := doc.AddDocToHelpSystem(hs); err != nil {
        return nil, fmt.Errorf("load durableobjects docs: %w", err)
    }
    commands := []cmds.Command{
        newDocsListCommand(hs),
        newDocsShowCommand(hs),
        newDocsServeCommand(hs, ctx),     // ctx carries Host/Config if needed
    }
    return &providerapi.CommandSet{Commands: commands}, nil
}
```

`CommandSetContext` is accepted (and its `Host`/`Config` are available) even though the minimal verb does not need them, because the app layer always passes it and future subcommands (e.g. reading companion assets) will.

#### 6.4.2 `list` (Glaze command)

```go
func newDocsListCommand(hs *help.HelpSystem) cmds.Command {
    return &docsListCommand{CommandDescription: cmds.NewCommandDescription("list",
        cmds.WithShort("List bundled go-go-objects help pages"),
    )}
}

func (c *docsListCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
    sections, err := c.hs.QuerySections("")
    if err != nil { return err }
    for _, s := range sections {
        if err := gp.AddRow(ctx, types.NewRow(
            types.MRP("slug", s.Slug),
            types.MRP("title", s.Title),
            types.MRP("sectionType", s.SectionType.String()),
            types.MRP("topics", strings.Join(s.Topics, ",")),
        )); err != nil { return err }
    }
    return nil
}
```

Because it is a Glaze command, output can be rendered as a table, JSON, CSV, etc. via standard Glazed flags — a free win.

#### 6.4.3 `show <slug>` (Writer command)

```go
func newDocsShowCommand(hs *help.HelpSystem) cmds.Command {
    return &docsShowCommand{CommandDescription: cmds.NewCommandDescription("show",
        cmds.WithShort("Print a bundled go-go-objects help page"),
        cmds.WithArguments(
            fields.New("slug", fields.TypeString, fields.WithRequired(true), fields.WithHelp("Help page slug, e.g. go-go-objects-js-api")),
        ),
    )}
}

func (c *docsShowCommand) RunIntoWriter(ctx context.Context, vals *values.Values, w io.Writer) error {
    var s struct{ Slug string `glazed:"slug"` }
    if err := vals.DecodeSectionInto(schema.DefaultSlug, &s); err != nil { return err }
    section, err := c.hs.GetSectionWithSlug(s.Slug)
    if err != nil || section == nil {
        return fmt.Errorf("no help page with slug %q", s.Slug)
    }
    fmt.Fprintf(w, "# %s\n\n%s\n", section.Title, section.Content)
    return nil
}
```

This intentionally reuses the exact lookup the docaccess glazed adapter uses (`GetSectionWithSlug`), so behavior is identical whether a page is reached via `docs show <slug>` or `help <slug>` (and, in goja-repl, `require("docs").bySlug(...)`).

#### 6.4.4 `serve` (Bare command)

```go
func newDocsServeCommand(hs *help.HelpSystem, _ providerapi.CommandSetContext) cmds.Command {
    return &docsServeCommand{CommandDescription: cmds.NewCommandDescription("serve",
        cmds.WithShort("Serve bundled go-go-objects docs over HTTP"),
        cmds.WithFlags(
            fields.New("addr", fields.TypeString, fields.WithDefault("127.0.0.1:8788"), fields.WithHelp("HTTP listen address")),
        ),
    )}
}

func (c *docsServeCommand) Run(ctx context.Context, vals *values.Values) error {
    var s struct{ Addr string `glazed:"addr"` }
    if err := vals.DecodeSectionInto(schema.DefaultSlug, &s); err != nil { return err }
    mux := http.NewServeMux()
    registerDocsHTTPHandlers(mux, c.hs)   // GET /docs lists slugs; GET /docs/{slug} returns JSON
    server := &http.Server{Addr: s.Addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
    // ... signal-aware ListenAndServe identical in shape to serve.go
}
```

The HTTP shape is intentionally simple and JSON-first so non-CLI consumers (and tests) can consume it:

```
GET /docs                       -> [{"slug","title","sectionType","summary"}, ...]
GET /docs/{slug}                -> {"title","body","topics","metadata"}
```

### 6.5 End-to-end flows after the change

**Flow A — `help <slug>` (bundle the docs):**

> The JavaScript `require("docs")` surface shown at the bottom of this diagram is available in `goja-repl` but is **not** wired into generated xgoja runtimes (the docaccess runtime registrar is not added by `pkg/xgoja/app/factory.go`). It is shown for completeness and tracked as future work.

```
buildspec: sources:[{kind:help, provider.go-go-objects-durableobjects/source:go-go-objects}]
   │  (xgoja generate)
   ▼
runtime plan SourcePlan{Kind:"help", Provider:"go-go-objects-durableobjects", Source:"go-go-objects"}
   │  (generated main: Register -> Package has HelpSource)
   ▼
installRootFramework -> loadConfiguredHelpSources
   │  ResolveHelpSource("go-go-objects-durableobjects","go-go-objects")
   │  helpSystem.LoadSectionsFromFS(docFS, ".")
   ▼
help_cmd.SetupCobraRootCommand(helpSystem, root)        # help <slug>  (works in xgoja)
   │  (goja-repl only — not wired in xgoja today:)
   ▼
docaccess glazed provider wraps helpSystem  ->  require("docs").bySlug(...)   # future work
```

Note the source **ID** that JavaScript sees (in goja-repl) is the buildspec `id` (e.g. `go-go-objects-help`), **not** the provider `Name`. Interns frequently trip here. `docs.sources()` always prints the authoritative IDs.

**Flow B — the `docs` verb (mounted command):**

```
buildspec: commands:[{type:provider.command-set, provider:go-go-objects-durableobjects, name:docs, mount:durableobjects}]
   │  (xgoja generate)
   ▼
runtime plan CommandPlan{Type:"provider.command-set", Provider:..., Name:"docs", Mount:"durableobjects"}
   │  (generated main: Register -> Package has CommandSetProvider{Name:"docs"})
   ▼
Host.AttachProviderCommands(root)
   │  ResolveCommandSetProvider("go-go-objects-durableobjects","docs")
   │  newDocsCommandSet(ctx) -> {list, show, serve}
   │  applyMountToCommands(..., "durableobjects")
   ▼
cobra: durableobjects docs list | show <slug> | serve
```

### 6.6 Full reference buildspec

A minimal binary that both bundles the docs and mounts the verb:

```yaml
schema: xgoja/v2
name: do-docs-demo
app:
  name: do-docs-demo
go:
  module: xgoja.generated/do-docs-demo
  version: "1.26"
workspace:
  mode: auto
providers:
  - id: go-go-objects-durableobjects
    import: github.com/go-go-golems/go-go-objects/pkg/xgoja/providers/durableobjects
    register: Register
runtime:
  modules:
    - provider: go-go-objects-durableobjects
      name: durableobjects
      as: durableobjects
sources:
  - id: go-go-objects-help            # bundle docs -> help <slug> + require("docs")
    kind: help
    from:
      provider:
        provider: go-go-objects-durableobjects
        source: go-go-objects
commands:
  - id: eval                          # gives JS access to require("docs")
    type: builtin.eval
  - id: go-go-objects-docs            # mount the docs verb
    type: provider.command-set
    provider: go-go-objects-durableobjects
    name: docs
    mount: durableobjects
artifacts:
  - id: binary
    type: binary
    output: dist/do-docs-demo
```

After `xgoja generate` + build:

```bash
./dist/do-docs-demo help go-go-objects-overview          # Flow A (cobra)
./dist/do-docs-demo eval 'require("docs").sources()'     # Flow A (JS)
./dist/do-docs-demo durableobjects docs list             # Flow B
./dist/do-docs-demo durableobjects docs show go-go-objects-js-api
./dist/do-docs-demo durableobjects docs serve --addr 127.0.0.1:8788
```

---

## 7. Decision records

### Decision D1 — Ship docs from the provider package, not from `cmd/`

- **Context:** Today the docs live at `cmd/go-go-objects/doc/`. Only the standalone binary embeds them. A provider cannot `go:embed` files outside its own module directory tree.
- **Options considered:** (a) keep docs in `cmd/` and have the provider import them (impossible: `go:embed` is directory-local); (b) duplicate the docs into the provider; (c) move the docs into the provider and have `cmd/` import them back.
- **Decision:** (c). Move the docs to `pkg/xgoja/providers/durableobjects/doc/`; the standalone binary imports that package.
- **Rationale:** Single source of truth; satisfies `go:embed` locality; both consumers share one set of files; matches the pattern already used by `go-go-goja/pkg/xgoja/doc`.
- **Consequences:** `cmd/go-go-objects/doc/` is removed; `cmd/go-go-objects/main.go` imports the provider's `doc` package. No external API changes. Must be validated by `go build ./...`.
- **Status:** proposed

### Decision D2 — Use `providerapi.HelpSource` (provider-side), not `docaccess/runtime.HelpSource` (registrar-side)

- **Context:** There are two `HelpSource` types (section 4.5). One carries an `fs.FS`, the other a built `*help.HelpSystem`.
- **Options considered:** (a) build a HelpSystem inside `Register` and hand it to the registrar type; (b) ship an `fs.FS` via the provider type and let the app build the HelpSystem.
- **Decision:** (b). Use `providerapi.HelpSource{Name, FS, Root}`.
- **Rationale:** Provider registration happens once, before any binary is generated; building a HelpSystem at registration time is premature and would duplicate `loadConfiguredHelpSources`. The provider type is the documented, tested integration point (example 09, `app/root_test.go`).
- **Consequences:** The docs only load in binaries that select the source in their buildspec (by design). The verb builds its own ephemeral HelpSystem from the same FS so it works regardless.
- **Status:** proposed

### Decision D3 — The `docs` verb builds its own `*help.HelpSystem`

- **Context:** A command provider runs after the root framework has installed the global help system, but the global system is not passed through `CommandSetContext`.
- **Options considered:** (a) thread the global HelpSystem into `CommandSetContext`; (b) have the verb build a fresh HelpSystem from the embedded FS.
- **Decision:** (b).
- **Rationale:** Requires no `go-go-goja` changes; the embedded FS is cheap to parse; keeps the verb self-contained and correct even when the buildspec selects the verb but not the help source. (a) would be a cross-repo change for a minor dedup.
- **Consequences:** If docs are edited, both the global HelpSystem and the verb's HelpSystem come from the same embedded bytes, so they cannot diverge within a binary.
- **Status:** proposed

### Decision D4 — Fate of the standalone `cmd/go-go-objects` binary

- **Context:** After D1, the binary's docs come from the provider. Its other job (serving Durable Objects) is already covered by the provider's `serve` command set.
- **Options considered:** (a) delete the binary; (b) keep it as a thin demo server; (c) keep it unchanged.
- **Decision:** Recommend (b): keep it as a minimal, dependency-light demo/development entry point, with docs loaded from the provider package.
- **Rationale:** AGENT.md says avoid backwards-compat shims unless asked; the binary is a demo, not a public API. Keeping a thin demo preserves the "try the counter in 30 seconds" workflow from `01-overview.md` without duplicating logic.
- **Consequences:** Requires maintainer sign-off. If deleted, update `01-overview.md` to point at `xgoja generate` instead.
- **Status:** proposed (maintainer to accept/reject)

### Decision D5 — Verb command set name and mount

- **Context:** The provider already mounts `serve` under `durableobjects`. Adding `docs` should not collide and should read naturally.
- **Options considered:** (a) `docs` under `durableobjects`; (b) a top-level `objects-docs`; (c) a nested `durableobjects docs`.
- **Decision:** (a): `CommandSetProvider{Name:"docs", DefaultMount:"durableobjects"}`. Mount is overridable per buildspec.
- **Rationale:** Symmetric with `serve`; both read as `durableobjects <serve|docs> ...`; discoverable via `--help`.
- **Consequences:** `durableobjects docs list/show/serve`. If a buildspec mounts at root, the commands become top-level `list/show/serve`, which is discouraged.
- **Status:** proposed

---

## 8. Implementation plan (phased, file-level)

Each phase is independently buildable and testable. Do not combine phases; commit after each.

### Phase 1 — Relocate and re-embed the docs

**Goal:** Single source of truth for the three Markdown pages.

1. Create `go-go-objects/pkg/xgoja/providers/durableobjects/doc/`.
2. `git mv` the three files from `cmd/go-go-objects/doc/` into the new `doc/` directory:
   - `01-overview.md`, `02-javascript-api.md`, `03-xgoja-provider.md`.
3. Create `doc/doc.go` per section 6.2 (`//go:embed *.md`, `AddDocToHelpSystem`, `HelpFS`).
4. Update `cmd/go-go-objects/main.go` to import `github.com/go-go-golems/go-go-objects/pkg/xgoja/providers/durableobjects/doc` and call `doc.AddDocToHelpSystem(helpSystem)`. Delete `cmd/go-go-objects/doc/`.

**Validate:**
```bash
cd go-go-objects
gofmt -w ./pkg/xgoja/providers/durableobjects/doc/doc.go ./cmd/go-go-objects/main.go
go build ./...
go test ./cmd/go-go-objects/... -count=1
go run ./cmd/go-go-objects help go-go-objects-overview   # docs still render from the new location
```

**Commit:** `refactor(go-go-objects): move Glazed docs into the durableobjects provider package`

### Phase 2 — Register the HelpSource

**Goal:** A generated binary can bundle the docs.

1. In `durableobjects.go`, import the new `doc` package and add the `providerapi.HelpSource{Name:"go-go-objects", FS, Root}` entry to `Register` (section 6.3).
2. Add a unit test mirroring `app/root_test.go:TestGeneratedRootLoadsProviderHelpSource`: build a registry with `durableobjectsprovider.Register`, assert `ResolveHelpSource(PackageID, "go-go-objects")` returns the FS and that `helpSystem.LoadSectionsFromFS(fs, root)` loads exactly the three slugs (`go-go-objects-overview`, `go-go-objects-js-api`, `go-go-objects-xgoja-provider`).

**Validate:**
```bash
go test ./pkg/xgoja/providers/durableobjects/... -count=1 -run HelpSource
```

**Commit:** `feat(durableobjects): register go-go-objects HelpSource for xgoja binaries`

### Phase 3 — Add the `docs` verb

**Goal:** A generated binary can list/show/serve the docs.

1. Create `pkg/xgoja/providers/durableobjects/docs.go` with `newDocsCommandSet` and the three commands (section 6.4).
2. Add the `providerapi.CommandSetProvider{Name:"docs", DefaultMount:"durableobjects", NewCommandSet:newDocsCommandSet}` entry to `Register`.
3. Add unit tests for each command:
   - `list`: assert it returns 3 rows with the expected slugs.
   - `show`: assert it returns the body for a known slug and an error for an unknown slug.
   - `serve`: use `httptest.NewServer`-style listener (`serveOnListener` pattern in `serve.go`) and assert `GET /docs` and `GET /docs/<slug>` return expected JSON.

**Validate:**
```bash
gofmt -w ./pkg/xgoja/providers/durableobjects/docs.go
go test ./pkg/xgoja/providers/durableobjects/... -count=1
```

**Commit:** `feat(durableobjects): add docs CommandSetProvider (list/show/serve)`

### Phase 4 — Optional: a docs-verb help page + example buildspec

**Goal:** Dogfooding and discoverability.

1. Add `doc/04-docs-verb.md` describing the verb (frontmatter `Slug: go-go-objects-docs-verb`, `SectionType: GeneralTopic`).
2. Add an example buildspec under `go-go-goja/examples/xgoja/` (e.g. `18-go-go-objects-docs`) modeled on examples 05 and 09, with a `Makefile` that generates, builds, and runs the smoke-test commands from section 6.6. (This requires the go-go-objects module to be resolvable from the example workspace; reuse the `replace`/workspace pattern from example 09.)

**Validate:**
```bash
cd go-go-goja/examples/xgoja/18-go-go-objects-docs && make smoke
```

**Commit:** `docs(durableobjects): add docs-verb help page and xgoja example`

### Phase 5 — Update standalone binary (per Decision D4)

**Goal:** Keep the demo working with docs from the provider (already partially done in Phase 1). Confirm `--help` still lists the docs and the counter demo still serves.

**Validate:**
```bash
go run ./cmd/go-go-objects --addr 127.0.0.1:8787 &
curl -s -X POST http://127.0.0.1:8787/rpc/COUNTER/global/increment -d '[1]'
```

**Commit:** `chore(go-go-objects): standalone binary uses provider docs`

---

## 9. Test strategy

### 9.1 Unit tests (Go)

| Target | What to assert |
| --- | --- |
| `doc.HelpFS()` / `AddDocToHelpSystem` | Loads exactly the three known slugs. |
| `Register` + `ResolveHelpSource` | Returns non-nil FS, root `"."`, and loads the three pages. |
| `Register` + `ResolveCommandSetProvider("docs")` | Returns the provider; `NewCommandSet` yields 3 commands named `list`/`show`/`serve`. |
| `docsListCommand` | Emits 3 rows; slugs match. |
| `docsShowCommand` | Known slug → non-empty body; unknown slug → error. |
| `docsServeCommand` | `GET /docs` → 200 + array; `GET /docs/<slug>` → 200 + object with `body`. |

Reuse the `providerapi` registry and `help.NewHelpSystem()` directly; no need to spin a generated binary for unit coverage.

### 9.2 Integration via an xgoja example

The strongest correctness signal is a generated binary. Example 18 (Phase 4) should `make smoke`:

```bash
./dist/do-docs-demo help go-go-objects-overview                      # exit 0, prints page
./dist/do-docs-demo durableobjects docs list --output json           # 3 rows
./dist/do-docs-demo durableobjects docs show go-go-objects-js-api    # prints body
./dist/do-docs-demo durableobjects docs serve --addr 127.0.0.1:8788 &
curl -sf http://127.0.0.1:8788/docs/go-go-objects-overview           # 200 JSON
```

### 9.3 Regression guards

- `go build ./...` from the `goja-durable-objects` workspace root (covers all three modules).
- `go test ./...` for go-go-objects.
- Re-run example 05 and example 09 `make smoke` to confirm no providerapi regression.

---

## 10. Risks, alternatives, and open questions

### Risks

- **Silent no-op when the buildspec omits the source.** Registering the HelpSource does nothing without a `kind: help` source stanza (section 4.8). Mitigation: the docs verb (Flow B) works without the source, and the example buildspec (section 6.6) shows both stanzas.
- **Source-ID vs provider-Name confusion in JS.** `require("docs").bySlug(...)` needs the runtime-plan source `ID`, not the provider `Name`. Mitigation: the docs guide and example always print `docs.sources()` first; the docs-verb help page calls this out.
- **Frontmatter drift.** If a moved Markdown file loses valid Glazed frontmatter, `LoadSectionsFromFS` skips it silently. Mitigation: Phase 2 unit test asserts all three slugs load.
- **Example workspace module resolution (Phase 4).** Pointing an example at the local go-go-objects module needs `replace`/workspace wiring like example 09. If fragile, defer Phase 4.

### Alternatives considered

- **Generate a HelpSystem in `Register` and register it via `docaccess/runtime.HelpSource`.** Rejected (D2): premature and would require per-binary wiring; the provider `HelpSource` is the supported path.
- **Expose docs only through `help <slug>` and skip the verb.** Rejected by the task: the user explicitly wants a verb that exposes the documentation. The verb also enables `serve` and structured `list`.
- **Make the verb read the global root HelpSystem.** Rejected (D3): would require a cross-repo change to thread the HelpSystem through `CommandSetContext`; the embedded-FS approach is local and equally correct.

### Open questions

1. Should `docs serve` require authentication or bind to localhost only? (Default: localhost, matching `serve.go`.)
2. Should the verb expose jsdoc/plugin docs in addition to Glazed help? (Default: no — keep it scoped to go-go-objects pages.)
3. Should the standalone `cmd/go-go-objects` binary be deleted in favor of `xgoja generate`? (Decision D4 — maintainer sign-off.)
4. **Wiring `require("docs")` into xgoja.** During implementation (example 18) we confirmed the docaccess runtime registrar is not added by `pkg/xgoja/app/factory.go`, so `require("docs")` resolves in `goja-repl` but not in generated xgoja eval/run/repl runtimes. Exposing provider HelpSources to JavaScript inside xgoja is a clean follow-up (add `docaccess/runtime.NewRegistrar` to the factory's module list, with HelpSources built from the loaded help system) and is tracked as a separate ticket. This does not affect the two delivered surfaces (`help <slug>` and the `docs` verb).

---

## 11. References

### Key files (go-go-goja, the mechanism — do not edit)

- `go-go-goja/pkg/docaccess/model.go` — data model (`Entry`, `EntryRef`, `Query`).
- `go-go-goja/pkg/docaccess/provider.go` — `Provider` interface.
- `go-go-goja/pkg/docaccess/hub.go` — `Hub` fan-out registry.
- `go-go-goja/pkg/docaccess/glazed/provider.go` — HelpSystem → docaccess adapter (`NewProvider`, `EntryKindHelpSection`).
- `go-go-goja/pkg/docaccess/runtime/registrar.go` — builds Hub + `docs` module; `RuntimeHubContextKey`.
- `go-go-goja/pkg/doc/15-docs-module-guide.md` — the `require("docs")` API reference.
- `go-go-goja/pkg/xgoja/providerapi/help.go` — `providerapi.HelpSource` (the hook we use).
- `go-go-goja/pkg/xgoja/providerapi/commands.go` — `CommandSetProvider`, `CommandSetContext`.
- `go-go-goja/pkg/xgoja/providerapi/provider_registry.go` — `Package`, `Register`, `ResolveHelpSource`, `ResolveCommandSetProvider`.
- `go-go-goja/pkg/xgoja/app/framework.go` — `installRootFramework`, `loadConfiguredHelpSources`.
- `go-go-goja/pkg/xgoja/app/command_providers.go` — `AttachProviderCommands`, `AttachCommandProvider`.
- `go-go-goja/pkg/xgoja/app/runtime_plan.go` — `RuntimePlan`, `SourcePlan`, `CommandPlan`.
- `go-go-goja/cmd/xgoja/internal/generate/templates/main.go.tmpl` — generated `main.go`.
- `go-go-goja/examples/xgoja/09-provider-shipped-help-docs/xgoja.yaml` — reference help-source buildspec.
- `go-go-goja/examples/xgoja/05-command-provider/xgoja.yaml` — reference command-provider buildspec.
- `go-go-goja/pkg/xgoja/testprovider/provider.go` — canonical `CommandSetProvider` shape (`NewFixtureCommandSet`).
- `go-go-goja/cmd/goja-repl/root.go` — reference: hand-built `runtime.HelpSource` wiring.

### Key files (go-go-objects, the changes GOJA-DO-003 makes)

- `go-go-objects/pkg/xgoja/providers/durableobjects/durableobjects.go` — `Register`, `PackageID`; add HelpSource + docs CommandSetProvider.
- `go-go-objects/pkg/xgoja/providers/durableobjects/doc/` — NEW: embedded docs + `AddDocToHelpSystem` + `HelpFS`.
- `go-go-objects/pkg/xgoja/providers/durableobjects/docs.go` — NEW: the docs verb.
- `go-go-objects/pkg/xgoja/providers/durableobjects/serve.go` — reference for listener/signal patterns reused by `docs serve`.
- `go-go-objects/cmd/go-go-objects/main.go` — switch to provider `doc` import.

### API quick reference

```go
// Register a help source (provider side)
providerapi.HelpSource{Name: "go-go-objects", Description: "...", FS: fs.FS, Root: "."}

// Resolve it (app side)
ps, ok := registry.ResolveHelpSource("go-go-objects-durableobjects", "go-go-objects")
helpSystem.LoadSectionsFromFS(ps.FS, ps.Root)

// Register a docs verb (provider side)
providerapi.CommandSetProvider{
    Name: "docs", DefaultMount: "durableobjects",
    NewCommandSet: func(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) { ... },
}

// Resolve + mount it (app side)
csp, _ := registry.ResolveCommandSetProvider("go-go-objects-durableobjects", "docs")
set, _ := csp.NewCommandSet(ctx)        // ctx is built by app.Host.AttachCommandProvider
```

```javascript
// JavaScript surface (requires a repl/eval command + the help source)
const docs = require("docs")
docs.sources()                                  // [{id:"go-go-objects-help", kind:"glazed-help", ...}]
docs.bySlug("go-go-objects-help", "go-go-objects-js-api").body
```

---

## 12. One-page checklist for the intern

- [ ] Read sections 4.1–4.7 until the five-stage pipeline is clear.
- [ ] Phase 1: move docs, re-embed, fix the standalone binary, `go build ./...`.
- [ ] Phase 2: add `HelpSource`, add the resolution unit test.
- [ ] Phase 3: add `docs.go` (`list`/`show`/`serve`), add unit tests.
- [ ] Phase 4 (optional): add `04-docs-verb.md` and example 18.
- [ ] Remember: register in Go **and** select in the buildspec; the JS source ID is the buildspec `id`.
- [ ] Remember: the verb builds its own HelpSystem from the same embedded FS (D3).
- [ ] Run `go test ./...` and the example `make smoke` before declaring done.
