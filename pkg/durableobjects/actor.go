package durableobjects

import (
	"context"
	"encoding/json"
	"errors"
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

type dispatchValueKind int

const (
	dispatchValueRPC dispatchValueKind = iota
	dispatchValueFetch
	dispatchValueAlarm
)

type dispatchValue struct {
	kind  dispatchValueKind
	value goja.Value
}

type promiseSnapshot struct {
	state  goja.PromiseState
	result goja.Value
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
	return a.withInterrupt(ctx, func(ctx context.Context) (Result, error) {
		raw, err := a.invokeDispatch(ctx, env)
		if err != nil {
			return Result{}, err
		}
		settled, err := a.awaitDispatchValue(ctx, raw)
		if err != nil {
			return Result{}, err
		}
		return a.convertDispatchValue(ctx, settled)
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

func (a *Actor) invokeDispatch(ctx context.Context, env Envelope) (dispatchValue, error) {
	ret, err := a.runtime.Owner.Call(ctx, "durable-object."+string(env.Kind), func(ctx context.Context, vm *goja.Runtime) (any, error) {
		return a.invokeDispatchOnOwner(ctx, vm, env)
	})
	if err != nil {
		return dispatchValue{}, preserveCodeOrWrap(CodeExecutionError, "execute durable object dispatch", err)
	}
	value, ok := ret.(dispatchValue)
	if !ok {
		return dispatchValue{}, coded(CodeExecutionError, "durable object dispatch returned unexpected result %T", ret)
	}
	return value, nil
}

func (a *Actor) invokeDispatchOnOwner(ctx context.Context, vm *goja.Runtime, env Envelope) (dispatchValue, error) {
	if a.instance == nil {
		return dispatchValue{}, coded(CodeExecutionError, "durable object instance is not initialized")
	}
	switch env.Kind {
	case KindRPC:
		return a.invokeRPC(vm, env.Method, env.ArgsJSON)
	case KindFetch:
		return a.invokeFetch(vm, env.Request)
	case KindAlarm:
		return a.invokeAlarm(vm)
	default:
		_ = ctx
		return dispatchValue{}, coded(CodeBadRequest, "unsupported durable object dispatch kind %q", env.Kind)
	}
}

func (a *Actor) invokeRPC(vm *goja.Runtime, method string, argsJSON []byte) (dispatchValue, error) {
	if method == "" {
		return dispatchValue{}, coded(CodeBadRequest, "rpc method is required")
	}
	var args []any
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			var wrapper struct {
				Args []any `json:"args"`
			}
			if err2 := json.Unmarshal(argsJSON, &wrapper); err2 != nil {
				return dispatchValue{}, wrap(CodeBadRequest, "decode rpc arguments", err)
			}
			args = wrapper.Args
		}
	}
	fnVal := a.instance.Get(method)
	fn, ok := goja.AssertFunction(fnVal)
	if !ok {
		return dispatchValue{}, coded(CodeMethodNotFound, "durable object method %q not found", method)
	}
	jsArgs := make([]goja.Value, len(args))
	for i, arg := range args {
		jsArgs[i] = vm.ToValue(arg)
	}
	value, err := fn(a.instance, jsArgs...)
	if err != nil {
		return dispatchValue{}, preserveCodeOrWrap(CodeExecutionError, "call durable object method", err)
	}
	return dispatchValue{kind: dispatchValueRPC, value: value}, nil
}

func (a *Actor) invokeFetch(vm *goja.Runtime, req *FetchRequest) (dispatchValue, error) {
	if req == nil {
		return dispatchValue{}, coded(CodeBadRequest, "fetch request is required")
	}
	fnVal := a.instance.Get("fetch")
	fn, ok := goja.AssertFunction(fnVal)
	if !ok {
		return dispatchValue{kind: dispatchValueFetch, value: vm.ToValue(map[string]any{"status": 404, "body": "fetch not implemented"})}, nil
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
		return dispatchValue{}, preserveCodeOrWrap(CodeExecutionError, "call durable object fetch", err)
	}
	return dispatchValue{kind: dispatchValueFetch, value: value}, nil
}

func (a *Actor) invokeAlarm(vm *goja.Runtime) (dispatchValue, error) {
	fnVal := a.instance.Get("alarm")
	fn, ok := goja.AssertFunction(fnVal)
	if !ok {
		return dispatchValue{kind: dispatchValueAlarm}, nil
	}
	value, err := fn(a.instance)
	if err != nil {
		return dispatchValue{}, preserveCodeOrWrap(CodeExecutionError, "call durable object alarm", err)
	}
	return dispatchValue{kind: dispatchValueAlarm, value: value}, nil
}

func (a *Actor) awaitDispatchValue(ctx context.Context, raw dispatchValue) (dispatchValue, error) {
	value, err := a.awaitValue(ctx, raw.value)
	if err != nil {
		return dispatchValue{}, err
	}
	raw.value = value
	return raw, nil
}

func (a *Actor) awaitValue(ctx context.Context, value goja.Value) (goja.Value, error) {
	if value == nil {
		return nil, nil
	}
	promise, ok := value.Export().(*goja.Promise)
	if !ok {
		return value, nil
	}
	for {
		select {
		case <-ctx.Done():
			return nil, timeoutOrContextError(ctx)
		default:
		}
		ret, err := a.runtime.Owner.Call(ctx, "durable-object.promise-state", func(_ context.Context, vm *goja.Runtime) (any, error) {
			return promiseSnapshot{state: promise.State(), result: promise.Result()}, nil
		})
		if err != nil {
			if ctxErr := timeoutOrContextError(ctx); ctxErr != nil {
				return nil, ctxErr
			}
			return nil, preserveCodeOrWrap(CodeExecutionError, "read durable object promise state", err)
		}
		snapshot, ok := ret.(promiseSnapshot)
		if !ok {
			return nil, coded(CodeExecutionError, "durable object promise state returned unexpected result %T", ret)
		}
		switch snapshot.state {
		case goja.PromiseStatePending:
			select {
			case <-ctx.Done():
				return nil, timeoutOrContextError(ctx)
			case <-time.After(5 * time.Millisecond):
			}
		case goja.PromiseStateRejected:
			return nil, a.promiseRejectedError(ctx, snapshot.result)
		case goja.PromiseStateFulfilled:
			return snapshot.result, nil
		default:
			return nil, coded(CodeExecutionError, "durable object promise has unknown state %d", snapshot.state)
		}
	}
}

func (a *Actor) convertDispatchValue(ctx context.Context, value dispatchValue) (Result, error) {
	switch value.kind {
	case dispatchValueRPC:
		return a.convertRPCResult(ctx, value.value)
	case dispatchValueFetch:
		return a.convertFetchResult(ctx, value.value)
	case dispatchValueAlarm:
		return Result{}, nil
	default:
		return Result{}, coded(CodeExecutionError, "unknown durable object dispatch value kind %d", value.kind)
	}
}

func (a *Actor) convertRPCResult(ctx context.Context, value goja.Value) (Result, error) {
	ret, err := a.runtime.Owner.Call(ctx, "durable-object.rpc-result", func(_ context.Context, vm *goja.Runtime) (any, error) {
		var exported any
		if value != nil && !goja.IsUndefined(value) {
			exported = value.Export()
		}
		payload, err := json.Marshal(exported)
		if err != nil {
			return nil, wrap(CodeExecutionError, "encode rpc result", err)
		}
		return payload, nil
	})
	if err != nil {
		return Result{}, preserveCodeOrWrap(CodeExecutionError, "convert durable object rpc result", err)
	}
	payload, ok := ret.([]byte)
	if !ok {
		return Result{}, coded(CodeExecutionError, "durable object rpc conversion returned unexpected result %T", ret)
	}
	return Result{ValueJSON: payload}, nil
}

func (a *Actor) convertFetchResult(ctx context.Context, value goja.Value) (Result, error) {
	ret, err := a.runtime.Owner.Call(ctx, "durable-object.fetch-result", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return fetchResponseFromValue(vm, value), nil
	})
	if err != nil {
		return Result{}, preserveCodeOrWrap(CodeExecutionError, "convert durable object fetch result", err)
	}
	response, ok := ret.(FetchResponse)
	if !ok {
		return Result{}, coded(CodeExecutionError, "durable object fetch conversion returned unexpected result %T", ret)
	}
	return Result{Response: &response}, nil
}

func fetchResponseFromValue(vm *goja.Runtime, value goja.Value) FetchResponse {
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
	return response
}

func (a *Actor) promiseRejectedError(ctx context.Context, value goja.Value) error {
	ret, err := a.runtime.Owner.Call(ctx, "durable-object.promise-rejection", func(_ context.Context, vm *goja.Runtime) (any, error) {
		return valueString(vm, value), nil
	})
	if err != nil {
		return preserveCodeOrWrap(CodeExecutionError, "format durable object promise rejection", err)
	}
	return coded(CodeExecutionError, "durable object promise rejected: %s", ret)
}

func valueString(vm *goja.Runtime, value goja.Value) string {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return "undefined"
	}
	obj := value.ToObject(vm)
	if obj != nil {
		if message := obj.Get("message"); message != nil && !goja.IsUndefined(message) && !goja.IsNull(message) {
			return message.String()
		}
	}
	return value.String()
}

func (a *Actor) withInterrupt(parent context.Context, fn func(context.Context) (Result, error)) (Result, error) {
	if parent == nil {
		parent = context.Background()
	}
	dispatchCtx, cancel := context.WithCancel(parent)
	if a.cpuTimeout > 0 {
		dispatchCtx, cancel = context.WithTimeout(parent, a.cpuTimeout)
	}
	defer cancel()
	if a.cpuTimeout <= 0 || a.runtime == nil || a.runtime.VM == nil {
		return fn(dispatchCtx)
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
	result, err := fn(dispatchCtx)
	if err != nil && interrupted.Load() {
		return Result{}, timeoutErr
	}
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func timeoutOrContextError(ctx context.Context) error {
	if ctx == nil || ctx.Err() == nil {
		return nil
	}
	if ctx.Err() == context.DeadlineExceeded {
		return coded(CodeTimeout, "durable object dispatch timed out")
	}
	return ctx.Err()
}

func preserveCodeOrWrap(code ErrorCode, message string, err error) error {
	if err == nil {
		return nil
	}
	if durableErr := durableErrorFrom(err); durableErr != nil {
		return durableErr
	}
	return wrap(code, message, err)
}

func durableErrorFrom(err error) *Error {
	var durableErr *Error
	if errors.As(err, &durableErr) {
		return durableErr
	}
	var exception *goja.Exception
	if errors.As(err, &exception) {
		if exported := exception.Value().Export(); exported != nil {
			if nestedErr, ok := exported.(error); ok && errors.As(nestedErr, &durableErr) {
				return durableErr
			}
		}
	}
	return nil
}
