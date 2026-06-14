package durableobjects

import (
	"context"
	"time"
)

type StorageFactory interface {
	Open(ctx context.Context, id ObjectID) (Storage, error)
}

type Storage interface {
	Get(ctx context.Context, key string) (any, bool, error)
	Put(ctx context.Context, key string, value any) error
	Delete(ctx context.Context, key string) (bool, error)
	List(ctx context.Context, prefix string, limit int) (map[string]any, error)
	Transaction(ctx context.Context, fn func(StorageTx) error) error
	SetAlarm(ctx context.Context, dueAt time.Time) error
	GetAlarm(ctx context.Context) (*time.Time, error)
	DeleteAlarm(ctx context.Context) error
	Close() error
}

type StorageTx interface {
	Get(ctx context.Context, key string) (any, bool, error)
	Put(ctx context.Context, key string, value any) error
	Delete(ctx context.Context, key string) (bool, error)
	List(ctx context.Context, prefix string, limit int) (map[string]any, error)
}

var _ StorageTx = Storage(nil)
