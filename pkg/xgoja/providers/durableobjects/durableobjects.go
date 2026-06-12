package durableobjectsprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/go-go-golems/go-go-objects/pkg/durableobjects"
	"gopkg.in/yaml.v3"
)

const PackageID = "go-go-objects-durableobjects"

const HostServiceKey = "go-go-objects.durableobjects.gateway"

type GatewayService struct {
	Manager *durableobjects.Manager
	Handler http.Handler
}

func Register(registry *providerapi.ProviderRegistry) error {
	capability := newCapability()
	return registry.Package(PackageID,
		providerapi.Module{
			Name:        "durableobjects",
			DefaultAs:   "durableobjects",
			Description: "Durable Objects manager RPC/fetch helpers backed by go-go-objects",
			TypeScript:  TypeScriptModule(),
			NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
				return capability.newModuleLoader(ctx.Host)
			},
		},
		providerapi.WithPackageCapability(capability),
	)
}

type settings struct {
	Enabled       bool   `glazed:"enabled"`
	StorageRoot   string `glazed:"storage-root"`
	BundlePath    string `glazed:"bundle-path"`
	ManifestPath  string `glazed:"manifest-path"`
	CPUTimeout    string `glazed:"cpu-timeout"`
	IdleTimeout   string `glazed:"idle-timeout"`
	AlarmInterval string `glazed:"alarm-interval"`
	IdleInterval  string `glazed:"idle-interval"`
}

type runtimeEntry struct {
	mu      sync.Mutex
	manager *durableobjects.Manager
	gateway http.Handler
	cancel  context.CancelFunc
}

type capability struct {
	mu      sync.Mutex
	entries map[*goja.Runtime]*runtimeEntry
}

func newCapability() *capability {
	return &capability{entries: map[*goja.Runtime]*runtimeEntry{}}
}

func (c *capability) CapabilityID() string { return "go-go-objects.durableobjects.config" }

func (c *capability) GlazedConfigSections(providerapi.SectionRequest) ([]schema.Section, error) {
	section, err := schema.NewSection(
		"durableobjects",
		"Durable Objects",
		schema.WithPrefix("durableobjects-"),
		schema.WithFields(
			fields.New("enabled", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Enable Durable Objects manager initialization")),
			fields.New("storage-root", fields.TypeString, fields.WithDefault("./var/durable-objects"), fields.WithHelp("SQLite storage root for Durable Objects")),
			fields.New("bundle-path", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Path to a CommonJS bundle exporting objects")),
			fields.New("manifest-path", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Path to a JSON/YAML Durable Objects manifest")),
			fields.New("cpu-timeout", fields.TypeString, fields.WithDefault("2s"), fields.WithHelp("Per-dispatch JavaScript CPU timeout")),
			fields.New("idle-timeout", fields.TypeString, fields.WithDefault("5m"), fields.WithHelp("Idle actor eviction timeout")),
			fields.New("alarm-interval", fields.TypeString, fields.WithDefault("1s"), fields.WithHelp("Background alarm scheduler interval; 0 disables the loop")),
			fields.New("idle-interval", fields.TypeString, fields.WithDefault("1m"), fields.WithHelp("Background idle evictor interval; 0 disables the loop")),
		),
	)
	if err != nil {
		return nil, err
	}
	return []schema.Section{section}, nil
}

func (c *capability) InitRuntimeFromSections(ctx context.Context, vals *values.Values, handle providerapi.RuntimeInitializerHandle) error {
	if handle == nil || handle.EngineRuntime() == nil || handle.EngineRuntime().VM == nil {
		return fmt.Errorf("durableobjects provider runtime handle is nil")
	}
	cfg := settings{Enabled: false, StorageRoot: "./var/durable-objects", CPUTimeout: "2s", IdleTimeout: "5m", AlarmInterval: "1s", IdleInterval: "1m"}
	if vals == nil {
		return nil
	}
	cfg.Enabled = true
	if err := vals.DecodeSectionInto("durableobjects", &cfg); err != nil {
		return err
	}
	if !cfg.Enabled {
		return nil
	}
	if cfg.BundlePath == "" {
		return fmt.Errorf("durableobjects bundle-path is required when enabled")
	}
	if cfg.ManifestPath == "" {
		return fmt.Errorf("durableobjects manifest-path is required when enabled")
	}

	manifest, err := loadManifest(cfg.ManifestPath)
	if err != nil {
		return err
	}
	bundleSource, err := os.ReadFile(cfg.BundlePath)
	if err != nil {
		return fmt.Errorf("read durableobjects bundle %q: %w", cfg.BundlePath, err)
	}
	cpuTimeout, err := parseDurationSetting("cpu-timeout", cfg.CPUTimeout)
	if err != nil {
		return err
	}
	idleTimeout, err := parseDurationSetting("idle-timeout", cfg.IdleTimeout)
	if err != nil {
		return err
	}
	manager, err := durableobjects.NewManager(
		manifest,
		durableobjects.NewBundle(string(bundleSource)),
		durableobjects.NewSQLiteStorageFactory(cfg.StorageRoot),
		durableobjects.Options{CPUTimeout: cpuTimeout, IdleTimeout: idleTimeout},
	)
	if err != nil {
		return err
	}

	runtime := handle.EngineRuntime()
	entry := c.entry(runtime.VM)
	entry.mu.Lock()
	entry.manager = manager
	entry.gateway = durableobjects.NewGateway(manager, durableobjects.GatewayOptions{DevErrors: true})
	entry.mu.Unlock()

	runtimeCtx, cancel := context.WithCancel(runtime.Context())
	entry.mu.Lock()
	entry.cancel = cancel
	entry.mu.Unlock()

	if alarmInterval, err := parseDurationSetting("alarm-interval", cfg.AlarmInterval); err != nil {
		return err
	} else if alarmInterval > 0 {
		go func() {
			_ = durableobjects.NewAlarmScheduler(manager, alarmInterval, 100, nil).Run(runtimeCtx)
		}()
	}
	if idleInterval, err := parseDurationSetting("idle-interval", cfg.IdleInterval); err != nil {
		return err
	} else if idleInterval > 0 {
		go func() {
			_ = durableobjects.NewIdleEvictor(manager, idleInterval, nil).Run(runtimeCtx)
		}()
	}

	return runtime.AddCloser(func(ctx context.Context) error {
		cancel()
		return c.shutdownRuntime(ctx, runtime.VM)
	})
}

func (c *capability) newModuleLoader(hostServices providerapi.HostServices) (require.ModuleLoader, error) {
	external, err := externalGatewayService(hostServices)
	if err != nil {
		return nil, err
	}
	return func(vm *goja.Runtime, moduleObj *goja.Object) {
		entry := c.entry(vm)
		entry.mu.Lock()
		manager := entry.manager
		if external.Manager != nil {
			manager = external.Manager
			entry.manager = manager
			entry.gateway = external.Handler
		}
		entry.mu.Unlock()

		exports := moduleObj.Get("exports").(*goja.Object)
		_ = exports.Set("rpc", func(namespace, name, method string, args []any) goja.Value {
			if manager == nil {
				panic(vm.NewGoError(fmt.Errorf("durableobjects manager is not initialized")))
			}
			id, err := durableobjects.NewObjectID(namespace, name)
			if err != nil {
				panic(vm.NewGoError(err))
			}
			payload, err := json.Marshal(args)
			if err != nil {
				panic(vm.NewGoError(err))
			}
			result, err := manager.Dispatch(context.Background(), durableobjects.Envelope{Kind: durableobjects.KindRPC, ID: id, Method: method, ArgsJSON: payload})
			if err != nil {
				panic(vm.NewGoError(err))
			}
			var value any
			if len(result.ValueJSON) > 0 {
				if err := json.Unmarshal(result.ValueJSON, &value); err != nil {
					panic(vm.NewGoError(err))
				}
			}
			return vm.ToValue(value)
		})
		_ = exports.Set("fetch", func(namespace, name string, request durableobjects.FetchRequest) goja.Value {
			if manager == nil {
				panic(vm.NewGoError(fmt.Errorf("durableobjects manager is not initialized")))
			}
			id, err := durableobjects.NewObjectID(namespace, name)
			if err != nil {
				panic(vm.NewGoError(err))
			}
			result, err := manager.Dispatch(context.Background(), durableobjects.Envelope{Kind: durableobjects.KindFetch, ID: id, Request: &request})
			if err != nil {
				panic(vm.NewGoError(err))
			}
			return vm.ToValue(result.Response)
		})
	}, nil
}

func (c *capability) entry(vm *goja.Runtime) *runtimeEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry := c.entries[vm]
	if entry == nil {
		entry = &runtimeEntry{}
		c.entries[vm] = entry
	}
	return entry
}

func (c *capability) shutdownRuntime(ctx context.Context, vm *goja.Runtime) error {
	c.mu.Lock()
	entry := c.entries[vm]
	delete(c.entries, vm)
	c.mu.Unlock()
	if entry == nil {
		return nil
	}
	entry.mu.Lock()
	cancel := entry.cancel
	manager := entry.manager
	entry.cancel = nil
	entry.manager = nil
	entry.gateway = nil
	entry.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if manager != nil {
		return manager.Close(ctx)
	}
	return nil
}

func externalGatewayService(hostServices providerapi.HostServices) (GatewayService, error) {
	lookup, ok := hostServices.(providerapi.HostServiceLookup)
	if !ok || lookup == nil {
		return GatewayService{}, nil
	}
	raw, ok := lookup.HostService(HostServiceKey)
	if !ok {
		return GatewayService{}, nil
	}
	service, ok := raw.(GatewayService)
	if !ok {
		return GatewayService{}, fmt.Errorf("durableobjects host service %q must be GatewayService, got %T", HostServiceKey, raw)
	}
	if service.Manager == nil {
		return GatewayService{}, fmt.Errorf("durableobjects host service %q has nil Manager", HostServiceKey)
	}
	return service, nil
}

func loadManifest(path string) (durableobjects.Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return durableobjects.Manifest{}, fmt.Errorf("read durableobjects manifest %q: %w", path, err)
	}
	var manifest durableobjects.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		if yamlErr := yaml.Unmarshal(data, &manifest); yamlErr != nil {
			return durableobjects.Manifest{}, fmt.Errorf("decode durableobjects manifest %q as JSON (%v) or YAML (%v)", path, err, yamlErr)
		}
	}
	if err := manifest.Validate(); err != nil {
		return durableobjects.Manifest{}, err
	}
	return manifest, nil
}

func parseDurationSetting(name, value string) (time.Duration, error) {
	if value == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse durableobjects %s duration %q: %w", name, value, err)
	}
	return d, nil
}

func TypeScriptModule() *spec.Module {
	return &spec.Module{
		Name: "durableobjects",
		RawDTS: []string{
			"export interface FetchRequest { method: string; url?: string; path: string; query?: Record<string, unknown>; headers?: Record<string, string>; body?: unknown; rawBody?: string }",
			"export interface FetchResponse { status: number; headers?: Record<string, string>; body?: unknown }",
			"export function rpc(namespace: string, name: string, method: string, args?: unknown[]): unknown;",
			"export function fetch(namespace: string, name: string, request: FetchRequest): FetchResponse;",
		},
	}
}

var _ providerapi.GlazedConfigSectionCapability = (*capability)(nil)
var _ providerapi.RuntimeInitializerCapability = (*capability)(nil)
