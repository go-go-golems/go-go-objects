package durableobjects

import (
	"context"
	"time"
)

type ErrorHandler func(error)

type AlarmScheduler struct {
	manager  *Manager
	interval time.Duration
	limit    int
	onError  ErrorHandler
}

func NewAlarmScheduler(manager *Manager, interval time.Duration, limit int, onError ErrorHandler) *AlarmScheduler {
	if interval <= 0 {
		interval = time.Second
	}
	if limit <= 0 {
		limit = 100
	}
	return &AlarmScheduler{manager: manager, interval: interval, limit: limit, onError: onError}
}

func (s *AlarmScheduler) Tick(ctx context.Context, now time.Time) (int, error) {
	if s == nil || s.manager == nil {
		return 0, coded(CodeExecutionError, "alarm scheduler manager is required")
	}
	return s.manager.DispatchDueAlarms(ctx, now, s.limit)
}

func (s *AlarmScheduler) Run(ctx context.Context) error {
	if s == nil || s.manager == nil {
		return coded(CodeExecutionError, "alarm scheduler manager is required")
	}
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			if _, err := s.Tick(ctx, now); err != nil && s.onError != nil {
				s.onError(err)
			}
		}
	}
}

type IdleEvictor struct {
	manager  *Manager
	interval time.Duration
	onError  ErrorHandler
}

func NewIdleEvictor(manager *Manager, interval time.Duration, onError ErrorHandler) *IdleEvictor {
	if interval <= 0 {
		interval = time.Second
	}
	return &IdleEvictor{manager: manager, interval: interval, onError: onError}
}

func (e *IdleEvictor) Tick(ctx context.Context, now time.Time) (int, error) {
	if e == nil || e.manager == nil {
		return 0, coded(CodeExecutionError, "idle evictor manager is required")
	}
	return e.manager.EvictIdle(ctx, now)
}

func (e *IdleEvictor) Run(ctx context.Context) error {
	if e == nil || e.manager == nil {
		return coded(CodeExecutionError, "idle evictor manager is required")
	}
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			if _, err := e.Tick(ctx, now); err != nil && e.onError != nil {
				e.onError(err)
			}
		}
	}
}
