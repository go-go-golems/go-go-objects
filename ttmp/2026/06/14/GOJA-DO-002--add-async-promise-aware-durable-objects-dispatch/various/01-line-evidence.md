---
Title: Line Evidence for Async Durable Objects Dispatch
Ticket: GOJA-DO-002
Status: active
Topics:
    - goja
    - durable-objects
    - actor-runtime
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: Source/evidence material for GOJA-DO-002 async Durable Objects dispatch design.
LastUpdated: 2026-06-14T18:00:00Z
WhatFor: Use as source evidence for Promise-aware Durable Objects dispatch design.
WhenToUse: Read when validating claims in the async dispatch design guide.
---

## go-go-objects/pkg/durableobjects/actor.go lines 25-218
    25	func (a *Actor) Dispatch(ctx context.Context, env Envelope) (Result, error) {
    26		if a == nil || a.runtime == nil || a.runtime.Owner == nil {
    27			return Result{}, coded(CodeExecutionError, "durable object actor is not initialized")
    28		}
    29		a.active.Add(1)
    30		a.touch()
    31		defer func() {
    32			a.touch()
    33			a.active.Add(-1)
    34		}()
    35		return a.withInterrupt(func() (Result, error) {
    36			ret, err := a.runtime.Owner.Call(ctx, "durable-object."+string(env.Kind), func(ctx context.Context, vm *goja.Runtime) (any, error) {
    37				return a.dispatchOnOwner(ctx, vm, env)
    38			})
    39			if err != nil {
    40				return Result{}, wrap(CodeExecutionError, "execute durable object dispatch", err)
    41			}
    42			result, ok := ret.(Result)
    43			if !ok {
    44				return Result{}, coded(CodeExecutionError, "durable object dispatch returned unexpected result %T", ret)
    45			}
    46			return result, nil
    47		})
    48	}
    49	
    50	func (a *Actor) Close(ctx context.Context) error {
    51		if a == nil || a.runtime == nil {
    52			return nil
    53		}
    54		return a.runtime.Close(ctx)
    55	}
    56	
    57	func (a *Actor) touch() {
    58		if a != nil {
    59			a.lastUsedNS.Store(time.Now().UnixNano())
    60		}
    61	}
    62	
    63	func (a *Actor) isIdle(now time.Time, idleTimeout time.Duration) bool {
    64		if a == nil || idleTimeout <= 0 || a.active.Load() > 0 {
    65			return false
    66		}
    67		last := a.lastUsedNS.Load()
    68		if last == 0 {
    69			return false
    70		}
    71		return now.Sub(time.Unix(0, last)) >= idleTimeout
    72	}
    73	
    74	func (a *Actor) bootstrap(ctx context.Context, bundle *Bundle) error {
    75		_, err := a.runtime.Owner.Call(ctx, "durable-object.bootstrap", func(ctx context.Context, vm *goja.Runtime) (any, error) {
    76			exports, err := bundle.Evaluate(ctx, vm)
    77			if err != nil {
    78				return nil, err
    79			}
    80			objects := exports.Get("objects")
    81			if objects == nil || goja.IsUndefined(objects) || goja.IsNull(objects) {
    82				return nil, coded(CodeExecutionError, "durable object bundle must export objects")
    83			}
    84			ctorVal := objects.ToObject(vm).Get(a.className)
    85			if ctorVal == nil || goja.IsUndefined(ctorVal) || goja.IsNull(ctorVal) {
    86				return nil, coded(CodeExecutionError, "durable object class %q not found in exports.objects", a.className)
    87			}
    88			ctor, ok := goja.AssertConstructor(ctorVal)
    89			if !ok {
    90				return nil, coded(CodeExecutionError, "durable object class %q is not a constructor", a.className)
    91			}
    92			instance, err := ctor(ctorVal.ToObject(vm), newStateObject(vm, a), newEnvObject(vm, a.manager, a.id))
    93			if err != nil {
    94				return nil, wrap(CodeExecutionError, "construct durable object instance", err)
    95			}
    96			a.instance = instance
    97			return nil, nil
    98		})
    99		return err
   100	}
   101	
   102	func (a *Actor) dispatchOnOwner(ctx context.Context, vm *goja.Runtime, env Envelope) (Result, error) {
   103		if a.instance == nil {
   104			return Result{}, coded(CodeExecutionError, "durable object instance is not initialized")
   105		}
   106		switch env.Kind {
   107		case KindRPC:
   108			return a.callRPC(vm, env.Method, env.ArgsJSON)
   109		case KindFetch:
   110			return a.callFetch(vm, env.Request)
   111		case KindAlarm:
   112			return a.callAlarm(vm)
   113		default:
   114			_ = ctx
   115			return Result{}, coded(CodeBadRequest, "unsupported durable object dispatch kind %q", env.Kind)
   116		}
   117	}
   118	
   119	func (a *Actor) callRPC(vm *goja.Runtime, method string, argsJSON []byte) (Result, error) {
   120		if method == "" {
   121			return Result{}, coded(CodeBadRequest, "rpc method is required")
   122		}
   123		var args []any
   124		if len(argsJSON) > 0 {
   125			if err := json.Unmarshal(argsJSON, &args); err != nil {
   126				var wrapper struct {
   127					Args []any `json:"args"`
   128				}
   129				if err2 := json.Unmarshal(argsJSON, &wrapper); err2 != nil {
   130					return Result{}, wrap(CodeBadRequest, "decode rpc arguments", err)
   131				}
   132				args = wrapper.Args
   133			}
   134		}
   135		fnVal := a.instance.Get(method)
   136		fn, ok := goja.AssertFunction(fnVal)
   137		if !ok {
   138			return Result{}, coded(CodeMethodNotFound, "durable object method %q not found", method)
   139		}
   140		jsArgs := make([]goja.Value, len(args))
   141		for i, arg := range args {
   142			jsArgs[i] = vm.ToValue(arg)
   143		}
   144		value, err := fn(a.instance, jsArgs...)
   145		if err != nil {
   146			return Result{}, wrap(CodeExecutionError, "call durable object method", err)
   147		}
   148		payload, err := json.Marshal(value.Export())
   149		if err != nil {
   150			return Result{}, wrap(CodeExecutionError, "encode rpc result", err)
   151		}
   152		return Result{ValueJSON: payload}, nil
   153	}
   154	
   155	func (a *Actor) callFetch(vm *goja.Runtime, req *FetchRequest) (Result, error) {
   156		if req == nil {
   157			return Result{}, coded(CodeBadRequest, "fetch request is required")
   158		}
   159		fnVal := a.instance.Get("fetch")
   160		fn, ok := goja.AssertFunction(fnVal)
   161		if !ok {
   162			return Result{Response: &FetchResponse{Status: 404, Body: "fetch not implemented"}}, nil
   163		}
   164		value, err := fn(a.instance, vm.ToValue(map[string]any{
   165			"method":  req.Method,
   166			"url":     req.URL,
   167			"path":    req.Path,
   168			"query":   req.Query,
   169			"headers": req.Headers,
   170			"body":    req.Body,
   171			"rawBody": req.RawBody,
   172		}))
   173		if err != nil {
   174			return Result{}, wrap(CodeExecutionError, "call durable object fetch", err)
   175		}
   176		obj := value.ToObject(vm)
   177		response := FetchResponse{Status: 200}
   178		if status := obj.Get("status"); status != nil && !goja.IsUndefined(status) && !goja.IsNull(status) {
   179			response.Status = int(status.ToInteger())
   180		}
   181		if headers := obj.Get("headers"); headers != nil && !goja.IsUndefined(headers) && !goja.IsNull(headers) {
   182			response.Headers = map[string]string{}
   183			headersObj := headers.ToObject(vm)
   184			for _, key := range headersObj.Keys() {
   185				response.Headers[key] = headersObj.Get(key).String()
   186			}
   187		}
   188		if body := obj.Get("body"); body != nil && !goja.IsUndefined(body) && !goja.IsNull(body) {
   189			response.Body = body.Export()
   190		}
   191		return Result{Response: &response}, nil
   192	}
   193	
   194	func (a *Actor) callAlarm(vm *goja.Runtime) (Result, error) {
   195		fnVal := a.instance.Get("alarm")
   196		fn, ok := goja.AssertFunction(fnVal)
   197		if !ok {
   198			return Result{}, nil
   199		}
   200		if _, err := fn(a.instance); err != nil {
   201			return Result{}, wrap(CodeExecutionError, "call durable object alarm", err)
   202		}
   203		return Result{}, nil
   204	}
   205	
   206	func (a *Actor) withInterrupt(fn func() (Result, error)) (Result, error) {
   207		if a.cpuTimeout <= 0 || a.runtime == nil || a.runtime.VM == nil {
   208			return fn()
   209		}
   210		timeoutErr := coded(CodeTimeout, "durable object CPU budget exceeded")
   211		var interrupted atomic.Bool
   212		timer := time.AfterFunc(a.cpuTimeout, func() {
   213			interrupted.Store(true)
   214			a.runtime.VM.Interrupt(timeoutErr)
   215		})
   216		defer func() {
   217			_ = timer.Stop()
   218			a.runtime.VM.ClearInterrupt()

## go-go-objects/pkg/durableobjects/modules.go lines 50-72
    50			values, err := storage.List(runtimebridge.CurrentOwnerContext(vm), prefix, limit)
    51			if err != nil {
    52				panic(vm.NewGoError(err))
    53			}
    54			return vm.ToValue(values)
    55		})
    56		_ = obj.Set("transaction", func(fn goja.Value) error {
    57			callable, ok := goja.AssertFunction(fn)
    58			if !ok {
    59				return coded(CodeBadRequest, "storage.transaction requires a callback")
    60			}
    61			return storage.Transaction(runtimebridge.CurrentOwnerContext(vm), func(tx StorageTx) error {
    62				txObj := newStorageTxObject(vm, tx)
    63				ret, err := callable(goja.Undefined(), txObj)
    64				if err != nil {
    65					return err
    66				}
    67				if promise, ok := ret.Export().(*goja.Promise); ok && promise.State() == goja.PromiseStatePending {
    68					return coded(CodeBadRequest, "storage.transaction callback must be synchronous")
    69				}
    70				return nil
    71			})
    72		})

## go-go-goja/pkg/replsession/evaluate.go lines 507-636
   507		if promise, ok := value.Export().(*goja.Promise); ok {
   508			if policy.Eval.SupportTopLevelAwait {
   509				outcome.Awaited = true
   510				value, err = s.waitPromise(execCtx, promise)
   511				if err != nil {
   512					return outcome, err
   513				}
   514			} else {
   515				outcome.LastValue = promisePreview(promise)
   516				outcome.LastValueJSON = s.resultEnvelopeJSON(ctx, value)
   517				return outcome, nil
   518			}
   519		}
   520		outcome.LastValue = gojaValuePreview(value, s.runtime.VM)
   521		if outcome.LastValue == "" && (value == nil || goja.IsUndefined(value) || goja.IsNull(value)) {
   522			outcome.LastValue = "undefined"
   523		}
   524		outcome.LastValueJSON = s.resultEnvelopeJSON(ctx, value)
   525		return outcome, nil
   526	}
   527	
   528	func (s *sessionState) executeWrapped(ctx context.Context, rewrite RewriteReport) (executionOutcome, error) {
   529		outcome := executionOutcome{}
   530		execCtx, cancel := evaluationContext(ctx, s.policy)
   531		defer cancel()
   532	
   533		value, err := s.runString(execCtx, rewrite.TransformedSource)
   534		if err != nil {
   535			return outcome, err
   536		}
   537		if value == nil {
   538			return outcome, nil
   539		}
   540		if promise, ok := value.Export().(*goja.Promise); ok {
   541			outcome.Awaited = true
   542			value, err = s.waitPromise(execCtx, promise)
   543			if err != nil {
   544				return outcome, err
   545			}
   546		}
   547		persisted, lastValue, lastValueJSON, helperError, err := s.persistWrappedReturn(ctx, value, rewrite.BindingHelperName, rewrite.LastHelperName)
   548		if err != nil {
   549			return outcome, err
   550		}
   551		outcome.PersistedNames = persisted
   552		outcome.LastValue = lastValue
   553		outcome.LastValueJSON = lastValueJSON
   554		outcome.HelperError = helperError
   555		return outcome, nil
   556	}
   557	
   558	func (s *sessionState) runString(ctx context.Context, source string) (goja.Value, error) {
   559		if ctx == nil {
   560			ctx = context.Background()
   561		}
   562		if err := evaluationContextError(ctx); err != nil {
   563			return nil, err
   564		}
   565	
   566		stopInterrupt := make(chan struct{})
   567		interrupted := make(chan bool, 1)
   568		go func() {
   569			wasInterrupted := false
   570			select {
   571			case <-ctx.Done():
   572				wasInterrupted = true
   573				cause := evaluationContextError(ctx)
   574				if cause == nil {
   575					cause = context.Canceled
   576				}
   577				s.runtime.VM.Interrupt(cause)
   578			case <-stopInterrupt:
   579			}
   580			interrupted <- wasInterrupted
   581		}()
   582	
   583		ret, err := s.runtime.Owner.Call(context.WithoutCancel(ctx), "replsession.run-string", func(_ context.Context, vm *goja.Runtime) (any, error) {
   584			return vm.RunString(source)
   585		})
   586		close(stopInterrupt)
   587		if <-interrupted {
   588			s.runtime.VM.ClearInterrupt()
   589		}
   590		if err != nil {
   591			return nil, err
   592		}
   593		if ret == nil {
   594			return nil, nil
   595		}
   596		value, ok := ret.(goja.Value)
   597		if !ok {
   598			return nil, fmt.Errorf("unexpected evaluation result type %T", ret)
   599		}
   600		return value, nil
   601	}
   602	
   603	func (s *sessionState) waitPromise(ctx context.Context, promise *goja.Promise) (goja.Value, error) {
   604		for {
   605			select {
   606			case <-ctx.Done():
   607				return nil, evaluationContextError(ctx)
   608			default:
   609			}
   610	
   611			ret, err := s.runtime.Owner.Call(ctx, "replsession.promise-state", func(_ context.Context, vm *goja.Runtime) (any, error) {
   612				return promiseSnapshot{State: promise.State(), Result: promise.Result()}, nil
   613			})
   614			if err != nil {
   615				if cause := evaluationContextError(ctx); cause != nil {
   616					return nil, cause
   617				}
   618				return nil, err
   619			}
   620			snapshot, ok := ret.(promiseSnapshot)
   621			if !ok {
   622				return nil, fmt.Errorf("unexpected promise snapshot type %T", ret)
   623			}
   624			switch snapshot.State {
   625			case goja.PromiseStatePending:
   626				select {
   627				case <-ctx.Done():
   628					return nil, evaluationContextError(ctx)
   629				case <-time.After(5 * time.Millisecond):
   630				}
   631				continue
   632			case goja.PromiseStateRejected:
   633				return nil, fmt.Errorf("promise rejected: %s", rejectionMessage(snapshot.Result, s.runtime.VM))
   634			case goja.PromiseStateFulfilled:
   635				return snapshot.Result, nil
   636			default:

## go-go-goja/pkg/repl/evaluators/javascript/evaluator.go lines 313-361
   313		if promise, ok := result.Export().(*goja.Promise); ok {
   314			return e.waitForPromise(ctx, promise)
   315		}
   316	
   317		return result.String(), nil
   318	}
   319	
   320	func (e *Evaluator) waitForPromise(ctx context.Context, promise *goja.Promise) (string, error) {
   321		if promise == nil {
   322			return "undefined", nil
   323		}
   324		if e.ownedRuntime == nil {
   325			return promiseString(promise)
   326		}
   327	
   328		for {
   329			select {
   330			case <-ctx.Done():
   331				return "", ctx.Err()
   332			default:
   333			}
   334	
   335			ret, err := e.ownedRuntime.Owner.Call(ctx, "javascript.promise-state", func(_ context.Context, vm *goja.Runtime) (any, error) {
   336				return promiseSnapshot{
   337					State:  promise.State(),
   338					Result: promise.Result(),
   339				}, nil
   340			})
   341			if err != nil {
   342				return "", err
   343			}
   344			snapshot, ok := ret.(promiseSnapshot)
   345			if !ok {
   346				return "", errors.Errorf("unexpected promise snapshot type %T", ret)
   347			}
   348			switch snapshot.State {
   349			case goja.PromiseStatePending:
   350				time.Sleep(5 * time.Millisecond)
   351				continue
   352			case goja.PromiseStateRejected:
   353				return "", errors.Errorf("Promise rejected: %s", valueString(snapshot.Result))
   354			case goja.PromiseStateFulfilled:
   355				return valueString(snapshot.Result), nil
   356			}
   357		}
   358	}
   359	
   360	type promiseSnapshot struct {
   361		State  goja.PromiseState

## go-go-goja/pkg/doc/03-async-patterns.md lines 22-105
    22	A goja runtime is single-threaded from JavaScript's point of view. Any operation that touches JavaScript values, calls JavaScript functions, or resolves Promises must happen on the runtime owner.
    23	
    24	For native modules loaded inside an `engine.Runtime`, use `pkg/runtimebridge.RuntimeServices`:
    25	
    26	```go
    27	runtimeServices, ok := runtimebridge.Lookup(vm)
    28	if !ok || runtimeServices.Owner == nil {
    29	    panic(vm.NewGoError(fmt.Errorf("module requires runtime services")))
    30	}
    31	```
    32	
    33	`RuntimeServices` gives module code:
    34	
    35	- `Owner`: serialized access to the VM;
    36	- `Loop`: the underlying event loop when low-level integration is unavoidable;
    37	- `Lifetime()`: the runtime-owned lifetime context;
    38	- helper methods that make context intent explicit.
    39	
    40	## Runtime contexts
    41	
    42	The runtime API deliberately separates several context meanings:
    43	
    44	| Context | Purpose | Typical API |
    45	| --- | --- | --- |
    46	| Startup context | Runtime construction and initializers | `engine.WithStartupContext(ctx)` |
    47	| Lifetime context | Runtime-owned resources after construction | `engine.WithLifetimeContext(ctx)`, `RuntimeServices.Lifetime()` |
    48	| Current owner-entry context | The context for the JavaScript/native callback currently running on owner | `runtimebridge.CurrentOwnerContext(vm)` |
    49	| Custom operation context | HTTP request, Discord event, hardware event, or other external operation | `CallWithCustomContext`, `PostWithCustomContext` |
    50	
    51	Create runtimes with explicit startup and lifetime contexts:
    52	
    53	```go
    54	rt, err := factory.NewRuntime(
    55	    engine.WithStartupContext(ctx),
    56	    engine.WithLifetimeContext(ctx),
    57	)
    58	```
    59	
    60	Use separate contexts when construction and runtime lifetime are different:
    61	
    62	```go
    63	rt, err := factory.NewRuntime(
    64	    engine.WithStartupContext(startupCtx),
    65	    engine.WithLifetimeContext(lifetimeCtx),
    66	)
    67	```
    68	
    69	## Choosing the right RuntimeServices helper
    70	
    71	| Situation | Use |
    72	| --- | --- |
    73	| JS-facing native function calls another JS callback synchronously | `CallWithCurrentContext(vm, op, fn)` |
    74	| JS-facing native function schedules a follow-up on owner | `PostWithCurrentContext(vm, op, fn)` |
    75	| Runtime-owned background work settles a Promise | `PostWithLifetimeContext(op, fn)` or `PostWithCustomContext(callCtx, op, fn)` |
    76	| External request/event has its own context | `CallWithCustomContext(ctx, op, fn)` / `PostWithCustomContext(ctx, op, fn)` |
    77	| Need current callback context only | `runtimebridge.CurrentOwnerContext(vm)` |
    78	| Need runtime lifetime only | `runtimebridge.LifetimeContext(vm)` or `services.Lifetime()` |
    79	
    80	`CallWithCustomContext` and `PostWithCustomContext` link custom contexts to runtime lifetime cancellation. `CallWithCustomContext` cancels its linked context after the owner call returns. `PostWithCustomContext` keeps the linked context alive until the posted callback has executed, then unregisters the lifetime cancellation hook.
    81	
    82	## Promise-based API pattern
    83	
    84	```go
    85	func installSleep(vm *goja.Runtime, exports *goja.Object) {
    86	    runtimeServices, ok := runtimebridge.Lookup(vm)
    87	    if !ok || runtimeServices.Owner == nil {
    88	        panic(vm.NewGoError(fmt.Errorf("timer module requires runtime services")))
    89	    }
    90	
    91	    _ = exports.Set("sleep", func(ms int64) goja.Value {
    92	        promise, resolve, reject := vm.NewPromise()
    93	        callCtx := runtimebridge.CurrentOwnerContext(vm)
    94	        runtimeCtx := runtimeServices.Lifetime()
    95	
    96	        go func() {
    97	            if ms < 0 {
    98	                _ = runtimeServices.PostWithCustomContext(callCtx, "timer.sleep.reject", func(context.Context, *goja.Runtime) {
    99	                    _ = reject(vm.ToValue("timer.sleep: duration must be >= 0"))
   100	                })
   101	                return
   102	            }
   103	
   104	            timer := time.NewTimer(time.Duration(ms) * time.Millisecond)
   105	            defer timer.Stop()

## Cloudflare rules lines 632-682
   632	### Understand how input and output gates work
   633	
   634	While Durable Objects are single-threaded, JavaScript's `async` / `await` can allow multiple 
   635	requests to interleave execution while a request waits for the result of an asynchronous operation. 
   636	Cloudflare's runtime uses **input gates** and **output gates** to prevent data races and ensure 
   637	correctness by default.
   638	
   639	**Input gates** block new events (incoming requests, fetch responses) while synchronous JavaScript 
   640	execution is in progress. Awaiting async operations like `fetch()` or KV storage methods opens the 
   641	input gate, allowing other requests to interleave. However, storage operations provide special 
   642	protection:
   643	
   644	- [JavaScript](#tab-panel-8104)
   645	- [TypeScript](#tab-panel-8105)
   646	
   647	```js
   648	import { DurableObject } from "cloudflare:workers";
   649	
   650	export class Counter extends DurableObject {
   651	  // This code is safe due to input gates
   652	  async increment() {
   653	    // While these storage operations execute, no other requests
   654	    // can interleave - input gate blocks new events
   655	    const value = (await this.ctx.storage.get("count")) ?? 0;
   656	    await this.ctx.storage.put("count", value + 1);
   657	    return value + 1;
   658	  }
   659	}
   660	```
   661	
   662	**Output gates** hold outgoing network messages (responses, fetch requests) until pending storage 
   663	writes complete. This ensures clients never see confirmation of data that has not been persisted:
   664	
   665	- [JavaScript](#tab-panel-8106)
   666	- [TypeScript](#tab-panel-8107)
   667	
   668	```js
   669	import { DurableObject } from "cloudflare:workers";
   670	
   671	export class ChatRoom extends DurableObject {
   672	  async sendMessage(userId, content) {
   673	    // Write to storage - don't need to await for correctness
   674	    this.ctx.storage.sql.exec(
   675	      "INSERT INTO messages (user_id, content, created_at) VALUES (?, ?, ?)",
   676	      userId,
   677	      content,
   678	      Date.now(),
   679	    );
   680	
   681	    // This response is held by the output gate until the write completes.
   682	    // The client only receives "Message sent" after data is safely persisted.

## Cloudflare state lines 37-96
    37	### waitUntil
    38	
    39	`waitUntil` is available on `DurableObjectState` for API compatibility with [Workers Runtime 
    40	APIs](https://developers.cloudflare.com/workers/runtime-apis/context/#waituntil).
    41	
    42	#### Parameters
    43	
    44	- A required promise of any type.
    45	
    46	#### Return values
    47	
    48	- None.
    49	
    50	### blockConcurrencyWhile
    51	
    52	`blockConcurrencyWhile` executes an async callback while blocking any other events from being 
    53	delivered to the Durable Object until the callback completes. This method guarantees ordering and 
    54	prevents concurrent requests. All events that were not explicitly initiated as part of the callback 
    55	itself will be blocked. Once the callback completes, all other events will be delivered.
    56	
    57	- `blockConcurrencyWhile` is commonly used within the constructor of the Durable Object class to 
    58	enforce initialization to occur before any requests are delivered.
    59	- Another use case is executing `async` operations based on the current state of the Durable Object 
    60	and using `blockConcurrencyWhile` to prevent that state from changing while yielding the event loop.
    61	- If the callback throws an exception, the object will be terminated and reset. This ensures that 
    62	the object cannot be left stuck in an uninitialized state if something fails unexpectedly.
    63	- To avoid this behavior, enclose the body of your callback in a `try...catch` block to ensure it 
    64	cannot throw an exception.
    65	
    66	To help mitigate deadlocks there is a 30 second timeout applied when executing the callback. If 
    67	this timeout is exceeded, the Durable Object will be reset. It is best practice to have the 
    68	callback do as little work as possible to improve overall request throughput to the Durable Object.
    69	
    70	- [JavaScript](#tab-panel-8072)
    71	- [Python](#tab-panel-8073)
    72	
    73	```js
    74	// Durable Object
    75	export class MyDurableObject extends DurableObject {
    76	  initialized = false;
    77	
    78	  constructor(ctx, env) {
    79	    super(ctx, env);
    80	
    81	    // blockConcurrencyWhile will ensure that initialized will always be true
    82	    this.ctx.blockConcurrencyWhile(async () => {
    83	      this.initialized = true;
    84	    });
    85	  }
    86	  ...
    87	}
    88	```
    89	
    90	#### Parameters
    91	
    92	- A required callback which returns a `Promise<T>`.
    93	
    94	#### Return values
    95	
    96	- A `Promise<T>` returned by the callback.

## Cloudflare alarms lines 101-114
   101	- ``alarm(alarmInfo `  Object  `)``: `  void  `
   102		- Called by the system when a scheduled alarm time is reached.
   103			- The optional parameter `alarmInfo` object has two properties:
   104			- `retryCount` `  number  `: The number of times this alarm event has been retried.
   105					- `isRetry` `  boolean  `: A boolean value to indicate if the alarm 
   106	has been retried. This value is `true` if this alarm event is a retry.
   107			- Only one instance of `alarm()` will ever run at a given time per Durable Object 
   108	instance.
   109			- The `alarm()` handler has guaranteed at-least-once execution and will be retried 
   110	upon failure using exponential backoff, starting at 2 second delays for up to 6 retries. This only 
   111	applies to the most recent `setAlarm()` call. Retries will be performed if the method fails with an 
   112	uncaught exception.
   113			- This method can be `async`.
   114	
