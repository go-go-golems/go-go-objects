package durableobjects

import (
	"context"
	"net/http"
	"time"
)

// Server wraps a Durable Objects manager, gateway, alarm scheduler, and idle
// evictor for embedders that want to mount the runtime into an existing
// http.Server or ServeMux.
type Server struct {
	Manager *Manager
	Handler http.Handler
	cancel  context.CancelFunc
}

type ServerOptions struct {
	Manifest        Manifest
	BundleSource    string
	StorageRoot     string
	Storage         StorageFactory
	CPUTimeout      time.Duration
	IdleTimeout     time.Duration
	AlarmInterval   time.Duration
	IdleInterval    time.Duration
	MaxRequestBytes int64
	DevErrors       bool
	EventHook       EventHook
	ErrorHandler    func(error)
}

func NewServer(ctx context.Context, opts ServerOptions) (*Server, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	storage := opts.Storage
	if storage == nil {
		storage = NewSQLiteStorageFactory(opts.StorageRoot)
	}
	mgr, err := NewManager(
		opts.Manifest,
		NewBundle(opts.BundleSource),
		storage,
		Options{StorageRoot: opts.StorageRoot, CPUTimeout: opts.CPUTimeout, IdleTimeout: opts.IdleTimeout, EventHook: opts.EventHook},
	)
	if err != nil {
		return nil, err
	}
	serverCtx, cancel := context.WithCancel(ctx)
	server := &Server{
		Manager: mgr,
		Handler: NewGateway(mgr, GatewayOptions{MaxRequestBytes: opts.MaxRequestBytes, DevErrors: opts.DevErrors}),
		cancel:  cancel,
	}
	if opts.AlarmInterval > 0 {
		go func() {
			if err := NewAlarmScheduler(mgr, opts.AlarmInterval, 100, opts.ErrorHandler).Run(serverCtx); err != nil && opts.ErrorHandler != nil {
				opts.ErrorHandler(err)
			}
		}()
	}
	if opts.IdleInterval > 0 {
		go func() {
			if err := NewIdleEvictor(mgr, opts.IdleInterval, opts.ErrorHandler).Run(serverCtx); err != nil && opts.ErrorHandler != nil {
				opts.ErrorHandler(err)
			}
		}()
	}
	return server, nil
}

func (s *Server) Mount(mux *http.ServeMux) {
	if s == nil || mux == nil || s.Handler == nil {
		return
	}
	mux.Handle("/rpc/", s.Handler)
	mux.Handle("/fetch/", s.Handler)
}

func (s *Server) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if s.cancel != nil {
		s.cancel()
	}
	if s.Manager != nil {
		return s.Manager.Close(ctx)
	}
	return nil
}
