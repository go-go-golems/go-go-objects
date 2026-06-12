package durableobjectsprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/gojahttp"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
	"github.com/go-go-golems/go-go-goja/pkg/tsgen/spec"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	httpprovider "github.com/go-go-golems/go-go-goja/pkg/xgoja/providers/http"
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
			Name:         "durableobjects",
			DefaultAs:    "durableobjects",
			Description:  "Durable Objects manager RPC/fetch helpers backed by go-go-objects",
			ConfigSchema: moduleConfigSchema(),
			TypeScript:   TypeScriptModule(),
			NewModuleFactory: func(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
				return capability.newModuleLoader(ctx)
			},
		},
		providerapi.WithPackageCapability(capability),
	)
}

type settings struct {
	Enabled       bool   `glazed:"enabled" json:"enabled"`
	StorageRoot   string `glazed:"storage-root" json:"storageRoot"`
	BundlePath    string `glazed:"bundle-path" json:"bundlePath"`
	ManifestPath  string `glazed:"manifest-path" json:"manifestPath"`
	BundleAsset   string `json:"bundleAsset"`
	ManifestAsset string `json:"manifestAsset"`
	CPUTimeout    string `glazed:"cpu-timeout" json:"cpuTimeout"`
	IdleTimeout   string `glazed:"idle-timeout" json:"idleTimeout"`
	AlarmInterval string `glazed:"alarm-interval" json:"alarmInterval"`
	IdleInterval  string `glazed:"idle-interval" json:"idleInterval"`
}

type runtimeEntry struct {
	mu             sync.Mutex
	manager        *durableobjects.Manager
	gateway        http.Handler
	gatewayMounted bool
	cancel         context.CancelFunc
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
			fields.New("manifest-path", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Optional path to a JSON/YAML namespace manifest; defaults to deriving CAMEL_CASE namespaces from exports.objects")),
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

func (c *capability) XGojaConfigSection(providerapi.SectionRequest, providerapi.ModuleDescriptor) (schema.Section, error) {
	return schema.NewSection(
		"durableobjects-xgoja",
		"Durable Objects xgoja config",
		schema.WithFields(
			fields.New("storageRoot", fields.TypeString, fields.WithDefault("./var/durable-objects")),
			fields.New("bundlePath", fields.TypeString, fields.WithDefault("")),
			fields.New("manifestPath", fields.TypeString, fields.WithDefault("")),
			fields.New("bundleAsset", fields.TypeString, fields.WithDefault("")),
			fields.New("manifestAsset", fields.TypeString, fields.WithDefault("")),
			fields.New("cpuTimeout", fields.TypeString, fields.WithDefault("2s")),
			fields.New("idleTimeout", fields.TypeString, fields.WithDefault("5m")),
			fields.New("alarmInterval", fields.TypeString, fields.WithDefault("1s")),
			fields.New("idleInterval", fields.TypeString, fields.WithDefault("1m")),
		),
	)
}

func (c *capability) XGojaConfigFromGlazed(_ context.Context, req providerapi.XGojaConfigRequest) (*values.SectionValues, error) {
	out, err := values.NewSectionValues(req.ConfigSection)
	if err != nil {
		return nil, err
	}
	if req.GlazedValues == nil {
		return out, nil
	}
	enabled, ok := req.GlazedValues.GetField("durableobjects", "enabled")
	if !ok || enabled.Value != true {
		return out, nil
	}
	copies := map[string]string{
		"storage-root":   "storageRoot",
		"bundle-path":    "bundlePath",
		"manifest-path":  "manifestPath",
		"cpu-timeout":    "cpuTimeout",
		"idle-timeout":   "idleTimeout",
		"alarm-interval": "alarmInterval",
		"idle-interval":  "idleInterval",
	}
	for publicName, configName := range copies {
		field, ok := req.GlazedValues.GetField("durableobjects", publicName)
		if !ok {
			continue
		}
		definition, ok := req.ConfigSection.GetDefinitions().Get(configName)
		if !ok {
			return nil, fmt.Errorf("durableobjects internal config field %s not found", configName)
		}
		if err := out.Fields.UpdateWithLog(configName, definition, field.Value, field.Log...); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func defaultSettings() settings {
	return settings{Enabled: false, StorageRoot: "./var/durable-objects", CPUTimeout: "2s", IdleTimeout: "5m", AlarmInterval: "1s", IdleInterval: "1m"}
}

func (c *capability) InitRuntimeFromSections(ctx context.Context, vals *values.Values, handle providerapi.RuntimeInitializerHandle) error {
	_ = ctx
	_ = vals
	if handle == nil || handle.EngineRuntime() == nil || handle.EngineRuntime().VM == nil {
		return fmt.Errorf("durableobjects provider runtime handle is nil")
	}
	// Configuration is unified through XGojaConfigSectionCapability and handled
	// during module setup, where HostServices and embedded assets are available.
	// Keep this initializer as a lifecycle cleanup hook for command paths that
	// still invoke RuntimeInitializerCapability after runtime creation.
	runtime := handle.EngineRuntime()
	return runtime.AddCloser(func(ctx context.Context) error {
		return c.shutdownRuntime(ctx, runtime.VM)
	})
}

func (c *capability) newModuleLoader(ctx providerapi.ModuleSetupContext) (require.ModuleLoader, error) {
	configuredCtx, configuredCancel := context.WithCancel(context.Background())
	configured, err := gatewayServiceFromModuleConfig(configuredCtx, ctx.Host, ctx.Config)
	if err != nil {
		configuredCancel()
		return nil, err
	}
	if configured.Manager != nil && ctx.AddCloser != nil {
		manager := configured.Manager
		if err := ctx.AddCloser(func(ctx context.Context) error {
			configuredCancel()
			return manager.Close(ctx)
		}); err != nil {
			configuredCancel()
			_ = manager.Close(context.Background())
			return nil, err
		}
	}
	external, err := externalGatewayService(ctx.Host)
	if err != nil {
		return nil, err
	}
	httpHost, err := externalHTTPHost(ctx.Host)
	if err != nil {
		return nil, err
	}
	return func(vm *goja.Runtime, moduleObj *goja.Object) {
		entry := c.entry(vm)
		entry.mu.Lock()
		manager := entry.manager
		if configured.Manager != nil {
			manager = configured.Manager
			entry.manager = manager
			entry.gateway = configured.Handler
		}
		if external.Manager != nil {
			manager = external.Manager
			entry.manager = manager
			entry.gateway = external.Handler
		}
		if manager != nil && httpHost != nil && !entry.gatewayMounted {
			if entry.gateway == nil {
				entry.gateway = durableobjects.NewGateway(manager, durableobjects.GatewayOptions{DevErrors: true})
			}
			mountGatewayOnHTTPHost(httpHost, entry.gateway)
			entry.gatewayMounted = true
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
			result, err := manager.Dispatch(runtimebridge.CurrentOwnerContext(vm), durableobjects.Envelope{Kind: durableobjects.KindRPC, ID: id, Method: method, ArgsJSON: payload})
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
			result, err := manager.Dispatch(runtimebridge.CurrentOwnerContext(vm), durableobjects.Envelope{Kind: durableobjects.KindFetch, ID: id, Request: &request})
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

func gatewayServiceFromModuleConfig(ctx context.Context, host providerapi.HostServices, data json.RawMessage) (GatewayService, error) {
	if len(data) == 0 || string(data) == "null" {
		return GatewayService{}, nil
	}
	cfg := defaultSettings()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return GatewayService{}, fmt.Errorf("decode durableobjects module config: %w", err)
	}
	if cfg.BundlePath == "" && cfg.BundleAsset == "" && cfg.ManifestPath == "" && cfg.ManifestAsset == "" {
		return GatewayService{}, nil
	}
	return newGatewayServiceFromSettings(ctx, host, cfg)
}

func newGatewayServiceFromSettings(ctx context.Context, host providerapi.HostServices, cfg settings) (GatewayService, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	bundleSource, err := loadBundleSource(host, cfg)
	if err != nil {
		return GatewayService{}, err
	}
	manifest, err := loadConfiguredManifest(host, cfg)
	if err != nil {
		return GatewayService{}, err
	}
	cpuTimeout, err := parseDurationSetting("cpu-timeout", cfg.CPUTimeout)
	if err != nil {
		return GatewayService{}, err
	}
	idleTimeout, err := parseDurationSetting("idle-timeout", cfg.IdleTimeout)
	if err != nil {
		return GatewayService{}, err
	}
	manager, err := durableobjects.NewManager(
		manifest,
		durableobjects.NewBundle(bundleSource),
		durableobjects.NewSQLiteStorageFactory(cfg.StorageRoot),
		durableobjects.Options{CPUTimeout: cpuTimeout, IdleTimeout: idleTimeout},
	)
	if err != nil {
		return GatewayService{}, err
	}
	service := GatewayService{Manager: manager, Handler: durableobjects.NewGateway(manager, durableobjects.GatewayOptions{DevErrors: true})}
	if alarmInterval, err := parseDurationSetting("alarm-interval", cfg.AlarmInterval); err != nil {
		_ = manager.Close(ctx)
		return GatewayService{}, err
	} else if alarmInterval > 0 {
		go func() {
			_ = durableobjects.NewAlarmScheduler(manager, alarmInterval, 100, nil).Run(ctx)
		}()
	}
	if idleInterval, err := parseDurationSetting("idle-interval", cfg.IdleInterval); err != nil {
		_ = manager.Close(ctx)
		return GatewayService{}, err
	} else if idleInterval > 0 {
		go func() {
			_ = durableobjects.NewIdleEvictor(manager, idleInterval, nil).Run(ctx)
		}()
	}
	return service, nil
}

func loadBundleSource(host providerapi.HostServices, cfg settings) (string, error) {
	bundlePath := strings.TrimSpace(cfg.BundlePath)
	bundleAsset := strings.TrimSpace(cfg.BundleAsset)
	if bundlePath != "" && bundleAsset != "" {
		return "", fmt.Errorf("durableobjects config cannot combine bundlePath and bundleAsset")
	}
	if bundleAsset != "" {
		data, err := readAsset(host, bundleAsset, "bundle")
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	if bundlePath == "" {
		return "", fmt.Errorf("durableobjects bundle-path is required when enabled")
	}
	data, err := os.ReadFile(bundlePath)
	if err != nil {
		return "", fmt.Errorf("read durableobjects bundle %q: %w", bundlePath, err)
	}
	return string(data), nil
}

func loadConfiguredManifest(host providerapi.HostServices, cfg settings) (durableobjects.Manifest, error) {
	manifestPath := strings.TrimSpace(cfg.ManifestPath)
	manifestAsset := strings.TrimSpace(cfg.ManifestAsset)
	if manifestPath != "" && manifestAsset != "" {
		return durableobjects.Manifest{}, fmt.Errorf("durableobjects config cannot combine manifestPath and manifestAsset")
	}
	if cfg.BundlePath != "" && manifestAsset != "" {
		return durableobjects.Manifest{}, fmt.Errorf("durableobjects config cannot combine bundlePath and manifestAsset")
	}
	if cfg.BundleAsset != "" && manifestPath != "" {
		return durableobjects.Manifest{}, fmt.Errorf("durableobjects config cannot combine bundleAsset and manifestPath")
	}
	if manifestAsset != "" {
		data, err := readAsset(host, manifestAsset, "manifest")
		if err != nil {
			return durableobjects.Manifest{}, err
		}
		return decodeManifestBytes("asset "+manifestAsset, data)
	}
	if manifestPath != "" {
		return loadManifest(manifestPath)
	}
	return durableobjects.Manifest{}, nil
}

func readAsset(host providerapi.HostServices, id, kind string) ([]byte, error) {
	if host == nil || host.AssetResolver() == nil {
		return nil, fmt.Errorf("durableobjects %s asset %q requires a host asset resolver", kind, id)
	}
	fsys, path, ok := host.AssetResolver().ResolveAsset(id)
	if !ok {
		return nil, fmt.Errorf("durableobjects %s asset %q was not found", kind, id)
	}
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("read durableobjects %s asset %q: %w", kind, id, err)
	}
	return data, nil
}

func (c *capability) ContributeHostServices(ctx context.Context, req providerapi.HostServiceContributionRequest, sink providerapi.HostServiceSink) error {
	_ = ctx
	if sink == nil || !selectionIncludesHTTP(req.Modules) || !selectionIncludesDurableObjects(req.Modules) {
		return nil
	}
	host := gojahttp.NewHost(gojahttp.HostOptions{Dev: true})
	return sink.AddHostService(httpprovider.HostServiceKey, httpprovider.ExternalHostService{Host: host, OwnsListen: true})
}

func selectionIncludesHTTP(modules []providerapi.ModuleDescriptor) bool {
	for _, module := range modules {
		if module.PackageID == httpprovider.PackageID {
			return true
		}
	}
	return false
}

func selectionIncludesDurableObjects(modules []providerapi.ModuleDescriptor) bool {
	for _, module := range modules {
		if module.PackageID == PackageID {
			return true
		}
	}
	return false
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

func externalHTTPHost(hostServices providerapi.HostServices) (*gojahttp.Host, error) {
	lookup, ok := hostServices.(providerapi.HostServiceLookup)
	if !ok || lookup == nil {
		return nil, nil
	}
	values := lookup.HostServiceValues(httpprovider.HostServiceKey)
	if len(values) == 0 {
		return nil, nil
	}
	for _, raw := range values {
		service, ok := raw.(httpprovider.ExternalHostService)
		if !ok {
			return nil, fmt.Errorf("http host service %q must be ExternalHostService, got %T", httpprovider.HostServiceKey, raw)
		}
		if service.Host != nil {
			return service.Host, nil
		}
	}
	return nil, fmt.Errorf("http host service %q has nil Host", httpprovider.HostServiceKey)
}

func mountGatewayOnHTTPHost(host *gojahttp.Host, handler http.Handler) {
	if host == nil || handler == nil {
		return
	}
	host.RegisterStaticHandler("/rpc", restorePrefixHandler("/rpc", handler))
	host.RegisterStaticHandler("/fetch", restorePrefixHandler("/fetch", handler))
}

func restorePrefixHandler(prefix string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clone := r.Clone(r.Context())
		if clone.URL != nil {
			urlCopy := *clone.URL
			urlCopy.Path = prefix + urlCopy.Path
			clone.URL = &urlCopy
		}
		handler.ServeHTTP(w, clone)
	})
}

func loadManifest(path string) (durableobjects.Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return durableobjects.Manifest{}, fmt.Errorf("read durableobjects manifest %q: %w", path, err)
	}
	return decodeManifestBytes(path, data)
}

func decodeManifestBytes(source string, data []byte) (durableobjects.Manifest, error) {
	var manifest durableobjects.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		if yamlErr := yaml.Unmarshal(data, &manifest); yamlErr != nil {
			return durableobjects.Manifest{}, fmt.Errorf("decode durableobjects manifest %q as JSON (%v) or YAML (%v)", source, err, yamlErr)
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

func moduleConfigSchema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "storageRoot": {"type": "string", "description": "SQLite storage root for Durable Objects"},
    "bundlePath": {"type": "string", "description": "Path to a CommonJS bundle exporting objects"},
    "manifestPath": {"type": "string", "description": "Optional path to a JSON/YAML namespace manifest"},
    "bundleAsset": {"type": "string", "description": "Embedded asset id for a CommonJS bundle exporting objects"},
    "manifestAsset": {"type": "string", "description": "Optional embedded asset id for a JSON/YAML namespace manifest"},
    "cpuTimeout": {"type": "string", "description": "Per-dispatch JavaScript CPU timeout"},
    "idleTimeout": {"type": "string", "description": "Idle actor eviction timeout"},
    "alarmInterval": {"type": "string", "description": "Background alarm scheduler interval; 0 disables the loop"},
    "idleInterval": {"type": "string", "description": "Background idle evictor interval; 0 disables the loop"}
  }
}`)
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
var _ providerapi.XGojaConfigSectionCapability = (*capability)(nil)
var _ providerapi.HostServiceContributionCapability = (*capability)(nil)
var _ providerapi.RuntimeInitializerCapability = (*capability)(nil)
