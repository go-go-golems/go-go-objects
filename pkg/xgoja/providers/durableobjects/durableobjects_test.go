package durableobjectsprovider

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dop251/goja"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

const testBundle = `
class Counter {
  constructor(state, env) { this.state = state; this.env = env; }
  increment(by) {
    const current = this.state.storage.get("count") || 0;
    const next = current + (by || 1);
    this.state.storage.put("count", next);
    return next;
  }
  fetch(req) {
    if (req.path === "/count") return { status: 200, body: String(this.state.storage.get("count") || 0) };
    return { status: 404, body: "not found" };
  }
}
exports.objects = { Counter };
`

func TestRegister(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	mod, ok := registry.ResolveModule(PackageID, "durableobjects")
	if !ok {
		t.Fatal("expected durableobjects module")
	}
	if mod.TypeScript == nil {
		t.Fatal("expected TypeScript descriptor")
	}
	caps, ok := registry.ResolvePackageCapabilities(PackageID)
	if !ok || len(caps) != 1 {
		t.Fatalf("capabilities = %#v ok=%v", caps, ok)
	}
}

func TestCapabilityProvidesConfigSection(t *testing.T) {
	sections, err := newCapability().GlazedConfigSections(providerapi.SectionRequest{})
	if err != nil {
		t.Fatalf("GlazedConfigSections() error = %v", err)
	}
	if len(sections) != 1 || sections[0].GetSlug() != "durableobjects" {
		t.Fatalf("sections = %#v", sections)
	}
	if sections[0].GetPrefix() != "durableobjects-" {
		t.Fatalf("prefix = %q", sections[0].GetPrefix())
	}
}

func TestRuntimeInitializerAndModuleRPC(t *testing.T) {
	ctx := context.Background()
	capability := newCapability()
	bundlePath, _ := writeBundleAndManifest(t)
	vals := durableObjectsValues(t, map[string]any{
		"enabled":        true,
		"storage-root":   t.TempDir(),
		"bundle-path":    bundlePath,
		"alarm-interval": "0",
		"idle-interval":  "0",
	})

	loader, err := capability.newModuleLoader(providerapi.ModuleSetupContext{})
	if err != nil {
		t.Fatalf("newModuleLoader() error = %v", err)
	}
	factory, err := engine.NewRuntimeFactoryBuilder(
		engine.WithImplicitDefaultRegistryModules(false),
		engine.WithDataOnlyDefaultRegistryModules(true),
	).WithModules(engine.NativeModuleRegistrar{ModuleName: "durableobjects", Loader: loader}).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	rt, err := factory.NewRuntime(engine.WithStartupContext(ctx), engine.WithLifetimeContext(ctx))
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	if err := capability.InitRuntimeFromSections(ctx, vals, testRuntimeHandle{rt: rt}); err != nil {
		t.Fatalf("InitRuntimeFromSections() error = %v", err)
	}

	ret, err := rt.Owner.Call(ctx, "durableobjects.rpc", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(`require("durableobjects").rpc("COUNTER", "global", "increment", [2])`)
	})
	if err != nil {
		t.Fatalf("rpc call error = %v", err)
	}
	if got := ret.(goja.Value).Export(); got != int64(2) && got != float64(2) && got != 2 {
		t.Fatalf("rpc result = %#v", got)
	}
}

func TestModuleConfigLoadsBundleFromEmbeddedAsset(t *testing.T) {
	ctx := context.Background()
	capability := newCapability()
	host := testAssetHost{files: fstest.MapFS{
		"objects.js": &fstest.MapFile{Data: []byte(testBundle)},
	}}
	config, err := json.Marshal(map[string]any{
		"storageRoot":   t.TempDir(),
		"bundleAsset":   "objects.js",
		"alarmInterval": "0",
		"idleInterval":  "0",
	})
	if err != nil {
		t.Fatal(err)
	}
	var closers []func(context.Context) error
	loader, err := capability.newModuleLoader(providerapi.ModuleSetupContext{
		Context: ctx,
		Config:  config,
		Host:    host,
		AddCloser: func(fn func(context.Context) error) error {
			closers = append(closers, fn)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("newModuleLoader() error = %v", err)
	}
	defer func() {
		for i := len(closers) - 1; i >= 0; i-- {
			_ = closers[i](context.Background())
		}
	}()
	factory, err := engine.NewRuntimeFactoryBuilder(
		engine.WithImplicitDefaultRegistryModules(false),
		engine.WithDataOnlyDefaultRegistryModules(true),
	).WithModules(engine.NativeModuleRegistrar{ModuleName: "durableobjects", Loader: loader}).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	rt, err := factory.NewRuntime(engine.WithStartupContext(ctx), engine.WithLifetimeContext(ctx))
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	defer func() { _ = rt.Close(context.Background()) }()

	ret, err := rt.Owner.Call(ctx, "durableobjects.rpc", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return vm.RunString(`require("durableobjects").rpc("COUNTER", "asset", "increment", [3])`)
	})
	if err != nil {
		t.Fatalf("rpc call error = %v", err)
	}
	if got := ret.(goja.Value).Export(); got != int64(3) && got != float64(3) && got != 3 {
		t.Fatalf("rpc result = %#v", got)
	}
}

func TestModuleConfigRejectsMixedPathAndAssetModes(t *testing.T) {
	capability := newCapability()
	config, err := json.Marshal(map[string]any{
		"bundlePath":  "objects.js",
		"bundleAsset": "objects.js",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = capability.newModuleLoader(providerapi.ModuleSetupContext{Config: config, Host: testAssetHost{}})
	if err == nil || !strings.Contains(err.Error(), "cannot combine bundlePath and bundleAsset") {
		t.Fatalf("expected mixed mode error, got %v", err)
	}
}

func TestRuntimeInitializerRequiresBundleWhenEnabled(t *testing.T) {
	capability := newCapability()
	vm := goja.New()
	vals := durableObjectsValues(t, map[string]any{"enabled": true})
	err := capability.InitRuntimeFromSections(context.Background(), vals, testRuntimeHandle{rt: &engine.Runtime{VM: vm}})
	if err == nil || !strings.Contains(err.Error(), "bundle-path") {
		t.Fatalf("expected bundle-path error, got %v", err)
	}
}

func writeBundleAndManifest(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "objects.js")
	manifestPath := filepath.Join(dir, "durableobjects.json")
	if err := os.WriteFile(bundlePath, []byte(testBundle), 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	manifest := map[string]any{"objects": map[string]string{"COUNTER": "Counter"}}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return bundlePath, manifestPath
}

func durableObjectsValues(t *testing.T, overrides map[string]any) *values.Values {
	t.Helper()
	sections, err := newCapability().GlazedConfigSections(providerapi.SectionRequest{})
	if err != nil {
		t.Fatalf("sections: %v", err)
	}
	section := sections[0]
	fieldValues := fields.NewFieldValues()
	for _, definition := range section.GetDefinitions().ToList() {
		if definition.Default != nil {
			fieldValues.Set(definition.Name, &fields.FieldValue{Definition: definition, Value: *definition.Default})
		}
	}
	for name, value := range overrides {
		definition, ok := section.GetDefinitions().Get(name)
		if !ok {
			t.Fatalf("unknown field %q", name)
		}
		fieldValues.Set(name, &fields.FieldValue{Definition: definition, Value: value})
	}
	sectionValues, err := values.NewSectionValues(section, values.WithFields(fieldValues))
	if err != nil {
		t.Fatalf("section values: %v", err)
	}
	return values.New(values.WithSectionValues("durableobjects", sectionValues))
}

type testAssetHost struct {
	files fstest.MapFS
}

func (h testAssetHost) AssetResolver() providerapi.AssetResolver { return h }

func (h testAssetHost) ResolveAsset(id string) (fs.FS, string, bool) {
	if h.files == nil {
		return nil, "", false
	}
	if _, err := fs.Stat(h.files, id); err != nil {
		return nil, "", false
	}
	return h.files, id, true
}

type testRuntimeHandle struct {
	rt *engine.Runtime
}

func (h testRuntimeHandle) EngineRuntime() *engine.Runtime { return h.rt }
func (h testRuntimeHandle) Close(context.Context) error    { return nil }
