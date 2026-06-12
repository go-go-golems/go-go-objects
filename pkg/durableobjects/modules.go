package durableobjects

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/runtimebridge"
)

func newStateObject(vm *goja.Runtime, actor *Actor) goja.Value {
	state := vm.NewObject()
	_ = state.Set("id", map[string]any{
		"namespace": actor.id.Namespace,
		"name":      actor.id.Name,
		"hash":      actor.id.Hash,
	})
	_ = state.Set("storage", newStorageObject(vm, actor.storage))
	return state
}

func newStorageObject(vm *goja.Runtime, storage Storage) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("get", func(key string) (any, error) {
		value, ok, err := storage.Get(runtimebridge.CurrentOwnerContext(vm), key)
		if err != nil || !ok {
			return nil, err
		}
		return value, nil
	})
	_ = obj.Set("put", func(key string, value any) error {
		return storage.Put(runtimebridge.CurrentOwnerContext(vm), key, value)
	})
	_ = obj.Set("delete", func(key string) (bool, error) {
		return storage.Delete(runtimebridge.CurrentOwnerContext(vm), key)
	})
	_ = obj.Set("list", func(call goja.FunctionCall) goja.Value {
		prefix := ""
		limit := 1000
		if len(call.Arguments) > 0 && !goja.IsUndefined(call.Argument(0)) && !goja.IsNull(call.Argument(0)) {
			if obj := call.Argument(0).ToObject(vm); obj != nil && obj.Get("prefix") != nil && !goja.IsUndefined(obj.Get("prefix")) {
				prefix = obj.Get("prefix").String()
				if l := obj.Get("limit"); l != nil && !goja.IsUndefined(l) && !goja.IsNull(l) {
					limit = int(l.ToInteger())
				}
			} else {
				prefix = call.Argument(0).String()
			}
		}
		values, err := storage.List(runtimebridge.CurrentOwnerContext(vm), prefix, limit)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(values)
	})
	_ = obj.Set("transaction", func(fn goja.Value) error {
		callable, ok := goja.AssertFunction(fn)
		if !ok {
			return coded(CodeBadRequest, "storage.transaction requires a callback")
		}
		return storage.Transaction(runtimebridge.CurrentOwnerContext(vm), func(tx StorageTx) error {
			txObj := newStorageTxObject(vm, tx)
			ret, err := callable(goja.Undefined(), txObj)
			if err != nil {
				return err
			}
			if promise, ok := ret.Export().(*goja.Promise); ok && promise.State() == goja.PromiseStatePending {
				return coded(CodeBadRequest, "storage.transaction callback must be synchronous")
			}
			return nil
		})
	})
	_ = obj.Set("setAlarm", func(timestampMs int64) error {
		return storage.SetAlarm(runtimebridge.CurrentOwnerContext(vm), time.UnixMilli(timestampMs))
	})
	_ = obj.Set("getAlarm", func() (*int64, error) {
		dueAt, err := storage.GetAlarm(runtimebridge.CurrentOwnerContext(vm))
		if err != nil || dueAt == nil {
			return nil, err
		}
		ms := dueAt.UnixMilli()
		return &ms, nil
	})
	_ = obj.Set("deleteAlarm", func() error {
		return storage.DeleteAlarm(runtimebridge.CurrentOwnerContext(vm))
	})
	return obj
}

func newStorageTxObject(vm *goja.Runtime, tx StorageTx) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("get", func(key string) (any, error) {
		value, ok, err := tx.Get(runtimebridge.CurrentOwnerContext(vm), key)
		if err != nil || !ok {
			return nil, err
		}
		return value, nil
	})
	_ = obj.Set("put", func(key string, value any) error {
		return tx.Put(runtimebridge.CurrentOwnerContext(vm), key, value)
	})
	_ = obj.Set("delete", func(key string) (bool, error) {
		return tx.Delete(runtimebridge.CurrentOwnerContext(vm), key)
	})
	_ = obj.Set("list", func(prefix string) (map[string]any, error) {
		return tx.List(runtimebridge.CurrentOwnerContext(vm), prefix, 1000)
	})
	return obj
}

func newEnvObject(vm *goja.Runtime, manager *Manager, caller ObjectID) goja.Value {
	env := vm.NewObject()
	for namespace := range manager.manifest.Objects {
		namespace := namespace
		nsObj := vm.NewObject()
		_ = nsObj.Set("getByName", func(name string) (*goja.Object, error) {
			id, err := NewObjectID(namespace, name)
			if err != nil {
				return nil, err
			}
			return newStubObject(vm, manager, caller, id), nil
		})
		_ = env.Set(namespace, nsObj)
	}
	return env
}

func newStubObject(vm *goja.Runtime, manager *Manager, caller, target ObjectID) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("rpc", func(method string, args []any) goja.Value {
		if caller == target {
			panic(vm.NewGoError(coded(CodeBadRequest, "synchronous self-rpc is not supported")))
		}
		payload, err := json.Marshal(args)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		result, err := manager.Dispatch(runtimebridge.CurrentOwnerContext(vm), Envelope{Kind: KindRPC, ID: target, Method: method, ArgsJSON: payload})
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
	_ = obj.Set("id", map[string]any{"namespace": target.Namespace, "name": target.Name, "hash": target.Hash})
	return obj
}

func ownerContext(vm *goja.Runtime) context.Context {
	return runtimebridge.CurrentOwnerContext(vm)
}
