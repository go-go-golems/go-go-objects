package durableobjectsprovider

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	bundlePath, manifestPath := writeBundleAndManifest(t)
	vals := durableObjectsValues(t, map[string]any{
		"enabled":        true,
		"storage-root":   t.TempDir(),
		"bundle-path":    bundlePath,
		"manifest-path":  manifestPath,
		"alarm-interval": "0",
		"idle-interval":  "0",
	})

	loader, err := capability.newModuleLoader(nil)
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

func TestRuntimeInitializerRequiresBundleAndManifestWhenEnabled(t *testing.T) {
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

type testRuntimeHandle struct {
	rt *engine.Runtime
}

func (h testRuntimeHandle) EngineRuntime() *engine.Runtime { return h.rt }
func (h testRuntimeHandle) Close(context.Context) error    { return nil }
