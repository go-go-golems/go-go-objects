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
}

type Manager struct {
	mu       sync.Mutex
	actors   map[ObjectID]*Actor
	manifest Manifest
	bundle   *Bundle
	storage  StorageFactory
	opts     Options

	ctx    context.Context
	cancel context.CancelFunc
}

func NewManager(manifest Manifest, bundle *Bundle, storage StorageFactory, opts Options) (*Manager, error) {
	if err := manifest.Validate(); err != nil {
		return nil, err
	}
	if bundle == nil || bundle.Source == "" {
		return nil, coded(CodeBadRequest, "durable object bundle is required")
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
	if env.ID.IsZero() {
		return Result{}, coded(CodeBadRequest, "dispatch object id is required")
	}
	if _, ok := m.manifest.ClassForNamespace(env.ID.Namespace); !ok {
		return Result{}, coded(CodeUnknownNamespace, "unknown durable object namespace %q", env.ID.Namespace)
	}
	actor, err := m.getOrStart(ctx, env.ID)
	if err != nil {
		return Result{}, err
	}
	return actor.Dispatch(ctx, env)
}

func (m *Manager) Evict(ctx context.Context, id ObjectID) error {
	m.mu.Lock()
	actor := m.actors[id]
	delete(m.actors, id)
	m.mu.Unlock()
	if actor == nil {
		return nil
	}
	return actor.Close(ctx)
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
	m.mu.Lock()
	actor := m.actors[id]
	m.mu.Unlock()
	if actor != nil {
		return actor, nil
	}

	started, err := m.startActor(ctx, id)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if existing := m.actors[id]; existing != nil {
		_ = started.Close(ctx)
		return existing, nil
	}
	m.actors[id] = started
	return started, nil
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
		if err := candidate.actor.Close(ctx); err != nil && ret == nil {
			ret = err
		}
		evicted++
	}
	return evicted, ret
}

func (m *Manager) DispatchDueAlarms(ctx context.Context, now time.Time, limit int) (int, error) {
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
		if _, err := m.Dispatch(ctx, Envelope{Kind: KindAlarm, ID: record.ID}); err != nil {
			return dispatched, err
		}
		if err := index.DeleteAlarmIndex(ctx, record.ID); err != nil {
			return dispatched, err
		}
		dispatched++
	}
	return dispatched, nil
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
	actor := &Actor{id: id, className: className, runtime: rt, storage: storage, manager: m, cpuTimeout: m.opts.CPUTimeout}
	actor.touch()
	_ = rt.AddCloser(func(context.Context) error { return storage.Close() })
	if err := actor.bootstrap(ctx, m.bundle); err != nil {
		_ = rt.Close(ctx)
		return nil, err
	}
	return actor, nil
}
