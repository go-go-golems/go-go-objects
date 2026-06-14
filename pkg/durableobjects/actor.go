package durableobjects

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/pkg/engine"
)

type Actor struct {
	id         ObjectID
	className  string
	runtime    *engine.Runtime
	instance   *goja.Object // owner-thread only; access through runtime.Owner.Call.
	storage    Storage
	manager    *Manager
	cpuTimeout time.Duration
	lastUsedNS atomic.Int64
	active     atomic.Int32
}

func (a *Actor) Dispatch(ctx context.Context, env Envelope) (Result, error) {
	if a == nil || a.runtime == nil || a.runtime.Owner == nil {
		return Result{}, coded(CodeExecutionError, "durable object actor is not initialized")
	}
	a.active.Add(1)
	a.touch()
	defer func() {
		a.touch()
		a.active.Add(-1)
	}()
	return a.withInterrupt(func() (Result, error) {
		ret, err := a.runtime.Owner.Call(ctx, "durable-object."+string(env.Kind), func(ctx context.Context, vm *goja.Runtime) (any, error) {
			return a.dispatchOnOwner(ctx, vm, env)
		})
		if err != nil {
			return Result{}, wrap(CodeExecutionError, "execute durable object dispatch", err)
		}
		result, ok := ret.(Result)
		if !ok {
			return Result{}, coded(CodeExecutionError, "durable object dispatch returned unexpected result %T", ret)
		}
		return result, nil
	})
}

func (a *Actor) Close(ctx context.Context) error {
	if a == nil || a.runtime == nil {
		return nil
	}
	return a.runtime.Close(ctx)
}

func (a *Actor) touch() {
	if a != nil {
		a.lastUsedNS.Store(time.Now().UnixNano())
	}
}

func (a *Actor) isIdle(now time.Time, idleTimeout time.Duration) bool {
	if a == nil || idleTimeout <= 0 || a.active.Load() > 0 {
		return false
	}
	last := a.lastUsedNS.Load()
	if last == 0 {
		return false
	}
	return now.Sub(time.Unix(0, last)) >= idleTimeout
}

func (a *Actor) bootstrap(ctx context.Context, bundle *Bundle) error {
	_, err := a.runtime.Owner.Call(ctx, "durable-object.bootstrap", func(ctx context.Context, vm *goja.Runtime) (any, error) {
		exports, err := bundle.Evaluate(ctx, vm)
		if err != nil {
			return nil, err
		}
		objects := exports.Get("objects")
		if objects == nil || goja.IsUndefined(objects) || goja.IsNull(objects) {
			return nil, coded(CodeExecutionError, "durable object bundle must export objects")
		}
		ctorVal := objects.ToObject(vm).Get(a.className)
		if ctorVal == nil || goja.IsUndefined(ctorVal) || goja.IsNull(ctorVal) {
			return nil, coded(CodeExecutionError, "durable object class %q not found in exports.objects", a.className)
		}
		ctor, ok := goja.AssertConstructor(ctorVal)
		if !ok {
			return nil, coded(CodeExecutionError, "durable object class %q is not a constructor", a.className)
		}
		instance, err := ctor(ctorVal.ToObject(vm), newStateObject(vm, a), newEnvObject(vm, a.manager, a.id))
		if err != nil {
			return nil, wrap(CodeExecutionError, "construct durable object instance", err)
		}
		a.instance = instance
		return nil, nil
	})
	return err
}

func (a *Actor) dispatchOnOwner(ctx context.Context, vm *goja.Runtime, env Envelope) (Result, error) {
	if a.instance == nil {
		return Result{}, coded(CodeExecutionError, "durable object instance is not initialized")
	}
	switch env.Kind {
	case KindRPC:
		return a.callRPC(vm, env.Method, env.ArgsJSON)
	case KindFetch:
		return a.callFetch(vm, env.Request)
	case KindAlarm:
		return a.callAlarm(vm)
	default:
		_ = ctx
		return Result{}, coded(CodeBadRequest, "unsupported durable object dispatch kind %q", env.Kind)
	}
}

func (a *Actor) callRPC(vm *goja.Runtime, method string, argsJSON []byte) (Result, error) {
	if method == "" {
		return Result{}, coded(CodeBadRequest, "rpc method is required")
	}
	var args []any
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			var wrapper struct {
				Args []any `json:"args"`
			}
			if err2 := json.Unmarshal(argsJSON, &wrapper); err2 != nil {
				return Result{}, wrap(CodeBadRequest, "decode rpc arguments", err)
			}
			args = wrapper.Args
		}
	}
	fnVal := a.instance.Get(method)
	fn, ok := goja.AssertFunction(fnVal)
	if !ok {
		return Result{}, coded(CodeMethodNotFound, "durable object method %q not found", method)
	}
	jsArgs := make([]goja.Value, len(args))
	for i, arg := range args {
		jsArgs[i] = vm.ToValue(arg)
	}
	value, err := fn(a.instance, jsArgs...)
	if err != nil {
		return Result{}, wrap(CodeExecutionError, "call durable object method", err)
	}
	payload, err := json.Marshal(value.Export())
	if err != nil {
		return Result{}, wrap(CodeExecutionError, "encode rpc result", err)
	}
	return Result{ValueJSON: payload}, nil
}

func (a *Actor) callFetch(vm *goja.Runtime, req *FetchRequest) (Result, error) {
	if req == nil {
		return Result{}, coded(CodeBadRequest, "fetch request is required")
	}
	fnVal := a.instance.Get("fetch")
	fn, ok := goja.AssertFunction(fnVal)
	if !ok {
		return Result{Response: &FetchResponse{Status: 404, Body: "fetch not implemented"}}, nil
	}
	value, err := fn(a.instance, vm.ToValue(map[string]any{
		"method":  req.Method,
		"url":     req.URL,
		"path":    req.Path,
		"query":   req.Query,
		"headers": req.Headers,
		"body":    req.Body,
		"rawBody": req.RawBody,
	}))
	if err != nil {
		return Result{}, wrap(CodeExecutionError, "call durable object fetch", err)
	}
	obj := value.ToObject(vm)
	response := FetchResponse{Status: 200}
	if status := obj.Get("status"); status != nil && !goja.IsUndefined(status) && !goja.IsNull(status) {
		response.Status = int(status.ToInteger())
	}
	if headers := obj.Get("headers"); headers != nil && !goja.IsUndefined(headers) && !goja.IsNull(headers) {
		response.Headers = map[string]string{}
		headersObj := headers.ToObject(vm)
		for _, key := range headersObj.Keys() {
			response.Headers[key] = headersObj.Get(key).String()
		}
	}
	if body := obj.Get("body"); body != nil && !goja.IsUndefined(body) && !goja.IsNull(body) {
		response.Body = body.Export()
	}
	return Result{Response: &response}, nil
}

func (a *Actor) callAlarm(vm *goja.Runtime) (Result, error) {
	fnVal := a.instance.Get("alarm")
	fn, ok := goja.AssertFunction(fnVal)
	if !ok {
		return Result{}, nil
	}
	if _, err := fn(a.instance); err != nil {
		return Result{}, wrap(CodeExecutionError, "call durable object alarm", err)
	}
	return Result{}, nil
}

func (a *Actor) withInterrupt(fn func() (Result, error)) (Result, error) {
	if a.cpuTimeout <= 0 || a.runtime == nil || a.runtime.VM == nil {
		return fn()
	}
	timeoutErr := coded(CodeTimeout, "durable object CPU budget exceeded")
	var interrupted atomic.Bool
	timer := time.AfterFunc(a.cpuTimeout, func() {
		interrupted.Store(true)
		a.runtime.VM.Interrupt(timeoutErr)
	})
	defer func() {
		_ = timer.Stop()
		a.runtime.VM.ClearInterrupt()
	}()
	result, err := fn()
	if err != nil && interrupted.Load() {
		return Result{}, timeoutErr
	}
	return result, err
}
