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
