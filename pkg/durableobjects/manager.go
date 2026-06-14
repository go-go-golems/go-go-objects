package durableobjects

import (
	"context"
	"sync"
	"time"

	"github.com/go-go-golems/go-go-goja/pkg/engine"
)

type Options struct {
	StorageRoot string
	CPUTimeout  time.Duration
	IdleTimeout time.Duration
	EventHook   EventHook
}

type Manager struct {
	mu       sync.Mutex
	actors   map[ObjectID]*Actor
	starts   map[ObjectID]*startCall
	manifest Manifest
	bundle   *Bundle
	storage  StorageFactory
	opts     Options

	ctx    context.Context
	cancel context.CancelFunc
}

type startCall struct {
	done  chan struct{}
	actor *Actor
	err   error
}

func NewManager(manifest Manifest, bundle *Bundle, storage StorageFactory, opts Options) (*Manager, error) {
	if bundle == nil || bundle.Source == "" {
		return nil, coded(CodeBadRequest, "durable object bundle is required")
	}
	if manifest.IsZero() {
		derived, err := bundle.DeriveManifest(context.Background())
		if err != nil {
			return nil, err
		}
		manifest = derived
	} else if err := manifest.Validate(); err != nil {
		return nil, err
	}
	if storage == nil {
		if opts.StorageRoot == "" {
			return nil, coded(CodeBadRequest, "storage factory or storage root is required")
		}
		storage = NewSQLiteStorageFactory(opts.StorageRoot)
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		actors:   map[ObjectID]*Actor{},
		starts:   map[ObjectID]*startCall{},
		manifest: manifest,
		bundle:   bundle,
		storage:  storage,
		opts:     opts,
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

func (m *Manager) Dispatch(ctx context.Context, env Envelope) (Result, error) {
	if m == nil {
		return Result{}, coded(CodeExecutionError, "durable object manager is nil")
	}
	started := time.Now()
	var dispatchErr error
	m.emit(Event{Name: EventDispatchStart, ID: env.ID, Kind: env.Kind, Method: env.Method})
	defer func() {
		m.emit(Event{Name: EventDispatchEnd, ID: env.ID, Kind: env.Kind, Method: env.Method, Duration: time.Since(started), Error: dispatchErr})
	}()
	if env.ID.IsZero() {
		dispatchErr = coded(CodeBadRequest, "dispatch object id is required")
		return Result{}, dispatchErr
	}
	if _, ok := m.manifest.ClassForNamespace(env.ID.Namespace); !ok {
		dispatchErr = coded(CodeUnknownNamespace, "unknown durable object namespace %q", env.ID.Namespace)
		return Result{}, dispatchErr
	}
	actor, err := m.getOrStart(ctx, env.ID)
	if err != nil {
		dispatchErr = err
		return Result{}, dispatchErr
	}
	result, err := actor.Dispatch(ctx, env)
	if err != nil {
		dispatchErr = err
		return Result{}, dispatchErr
	}
	return result, nil
}

func (m *Manager) emit(event Event) {
	if m == nil || m.opts.EventHook == nil {
		return
	}
	m.opts.EventHook(event)
}

func (m *Manager) Evict(ctx context.Context, id ObjectID) error {
	m.mu.Lock()
	actor := m.actors[id]
	delete(m.actors, id)
	m.mu.Unlock()
	if actor == nil {
		return nil
	}
	m.emit(Event{Name: EventEvict, ID: id})
	err := actor.Close(ctx)
	m.emit(Event{Name: EventActorStop, ID: id, Error: err})
	return err
}

func (m *Manager) Close(ctx context.Context) error {
	if m.cancel != nil {
		m.cancel()
	}
	m.mu.Lock()
	actors := make([]*Actor, 0, len(m.actors))
	for id, actor := range m.actors {
		actors = append(actors, actor)
		delete(m.actors, id)
	}
	m.mu.Unlock()
	var ret error
	for _, actor := range actors {
		if err := actor.Close(ctx); err != nil && ret == nil {
			ret = err
		}
	}
	return ret
}

func (m *Manager) ActiveCount() int {
	if m == nil {
		return 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.actors)
}

func (m *Manager) getOrStart(ctx context.Context, id ObjectID) (*Actor, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	m.mu.Lock()
	if actor := m.actors[id]; actor != nil {
		m.mu.Unlock()
		return actor, nil
	}
	if call := m.starts[id]; call != nil {
		m.mu.Unlock()
		select {
		case <-call.done:
			return call.actor, call.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	call := &startCall{done: make(chan struct{})}
	m.starts[id] = call
	m.mu.Unlock()

	actor, err := m.startActor(ctx, id)
	var actorToClose *Actor
	m.mu.Lock()
	if err == nil {
		if existing := m.actors[id]; existing != nil {
			actorToClose = actor
			actor = existing
		} else {
			m.actors[id] = actor
		}
	}
	call.actor = actor
	call.err = err
	delete(m.starts, id)
	close(call.done)
	m.mu.Unlock()
	if actorToClose != nil {
		_ = actorToClose.Close(ctx)
	}
	return actor, err
}

func (m *Manager) EvictIdle(ctx context.Context, now time.Time) (int, error) {
	if m.opts.IdleTimeout <= 0 {
		return 0, nil
	}
	m.mu.Lock()
	var candidates []struct {
		id    ObjectID
		actor *Actor
	}
	for id, actor := range m.actors {
		if actor.isIdle(now, m.opts.IdleTimeout) {
			candidates = append(candidates, struct {
				id    ObjectID
				actor *Actor
			}{id: id, actor: actor})
		}
	}
	for _, candidate := range candidates {
		if m.actors[candidate.id] == candidate.actor {
			delete(m.actors, candidate.id)
		}
	}
	m.mu.Unlock()

	evicted := 0
	var ret error
	for _, candidate := range candidates {
		m.emit(Event{Name: EventEvict, ID: candidate.id})
		closeErr := candidate.actor.Close(ctx)
		if closeErr != nil && ret == nil {
			ret = closeErr
		}
		m.emit(Event{Name: EventActorStop, ID: candidate.id, Error: closeErr})
		evicted++
	}
	return evicted, ret
}

func (m *Manager) DispatchDueAlarms(ctx context.Context, now time.Time, limit int) (int, error) {
	if reconciler, ok := m.storage.(AlarmReconciler); ok {
		if _, err := reconciler.ReconcileAlarmIndex(ctx); err != nil {
			return 0, err
		}
	}
	index, ok := m.storage.(AlarmIndexer)
	if !ok {
		return 0, coded(CodeStorageError, "storage factory does not support alarm indexing")
	}
	due, err := index.DueAlarms(ctx, now, limit)
	if err != nil {
		return 0, err
	}
	dispatched := 0
	for _, record := range due {
		if err := m.clearAlarmBeforeDispatch(ctx, record.ID); err != nil {
			return dispatched, err
		}
		m.emit(Event{Name: EventAlarmDispatch, ID: record.ID})
		if _, err := m.Dispatch(ctx, Envelope{Kind: KindAlarm, ID: record.ID}); err != nil {
			return dispatched, err
		}
		dispatched++
	}
	return dispatched, nil
}

func (m *Manager) clearAlarmBeforeDispatch(ctx context.Context, id ObjectID) error {
	storage, err := m.storage.Open(ctx, id)
	if err != nil {
		return err
	}
	defer func() { _ = storage.Close() }()
	return storage.DeleteAlarm(ctx)
}

func (m *Manager) startActor(ctx context.Context, id ObjectID) (*Actor, error) {
	className, ok := m.manifest.ClassForNamespace(id.Namespace)
	if !ok {
		return nil, coded(CodeUnknownNamespace, "unknown durable object namespace %q", id.Namespace)
	}
	storage, err := m.storage.Open(ctx, id)
	if err != nil {
		return nil, err
	}

	factory, err := engine.NewRuntimeFactoryBuilder(
		engine.WithImplicitDefaultRegistryModules(false),
		engine.WithDataOnlyDefaultRegistryModules(true),
	).Build()
	if err != nil {
		_ = storage.Close()
		return nil, wrap(CodeActorStartFailed, "build actor runtime factory", err)
	}
	rt, err := factory.NewRuntime(engine.WithLifetimeContext(m.ctx))
	if err != nil {
		_ = storage.Close()
		return nil, wrap(CodeActorStartFailed, "create actor runtime", err)
	}
	actor := &Actor{id: id, className: className, runtime: rt, storage: storage, manager: m, cpuTimeout: m.opts.CPUTimeout, dispatchGate: make(chan struct{}, 1)}
	actor.touch()
	_ = rt.AddCloser(func(context.Context) error { return storage.Close() })
	if err := actor.bootstrap(ctx, m.bundle); err != nil {
		_ = rt.Close(ctx)
		return nil, err
	}
	m.emit(Event{Name: EventActorStart, ID: id})
	return actor, nil
}
