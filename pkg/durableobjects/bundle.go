package durableobjects

import (
	"context"
	"fmt"

	"github.com/dop251/goja"
)

type Bundle struct {
	Source string
}

func NewBundle(source string) *Bundle {
	return &Bundle{Source: source}
}

func (b *Bundle) DeriveManifest(ctx context.Context) (Manifest, error) {
	vm := goja.New()
	exports, err := b.Evaluate(ctx, vm)
	if err != nil {
		return Manifest{}, err
	}
	objectsValue := exports.Get("objects")
	objects := objectsValue.ToObject(vm)
	manifest := Manifest{Objects: map[string]string{}}
	for _, exportName := range objects.Keys() {
		ctor, ok := goja.AssertConstructor(objects.Get(exportName))
		if !ok || ctor == nil {
			return Manifest{}, coded(CodeExecutionError, "exports.objects.%s is not a constructor", exportName)
		}
		namespace := ExportNameToNamespace(exportName)
		if namespace == "" {
			return Manifest{}, coded(CodeExecutionError, "exports.objects contains invalid namespace key %q", exportName)
		}
		if existing, ok := manifest.Objects[namespace]; ok {
			return Manifest{}, coded(CodeExecutionError, "exports.objects keys %q and %q both map to namespace %q", existing, exportName, namespace)
		}
		manifest.Objects[namespace] = exportName
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func (b *Bundle) Evaluate(ctx context.Context, vm *goja.Runtime) (*goja.Object, error) {
	_ = ctx
	if b == nil || b.Source == "" {
		return nil, coded(CodeBadRequest, "durable object JavaScript bundle source is required")
	}
	exports := vm.NewObject()
	module := vm.NewObject()
	if err := module.Set("exports", exports); err != nil {
		return nil, err
	}
	if err := vm.Set("exports", exports); err != nil {
		return nil, err
	}
	if err := vm.Set("module", module); err != nil {
		return nil, err
	}
	if _, err := vm.RunString(b.Source); err != nil {
		return nil, wrap(CodeExecutionError, "evaluate durable object bundle", err)
	}
	moduleExports := module.Get("exports")
	if moduleExports == nil || goja.IsUndefined(moduleExports) || goja.IsNull(moduleExports) {
		return nil, coded(CodeExecutionError, "durable object bundle did not set module.exports")
	}
	obj := moduleExports.ToObject(vm)
	objects := obj.Get("objects")
	if objects == nil || goja.IsUndefined(objects) || goja.IsNull(objects) {
		return nil, fmt.Errorf("durable object bundle must export objects")
	}
	return obj, nil
}
