package durableobjects

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStorageFactory struct {
	Root string
}

type AlarmRecord struct {
	ID    ObjectID
	DueAt time.Time
}

func NewSQLiteStorageFactory(root string) *SQLiteStorageFactory {
	return &SQLiteStorageFactory{Root: root}
}

func (f *SQLiteStorageFactory) Open(ctx context.Context, id ObjectID) (Storage, error) {
	if f == nil || strings.TrimSpace(f.Root) == "" {
		return nil, coded(CodeBadRequest, "sqlite storage root is required")
	}
	if id.IsZero() {
		return nil, coded(CodeBadRequest, "object id is required")
	}
	path := f.pathFor(id)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, wrap(CodeStorageError, "create object storage directory", err)
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, wrap(CodeStorageError, "open object sqlite database", err)
	}
	s := &SQLiteStorage{db: db, path: path, id: id, factory: f}
	if err := s.init(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (f *SQLiteStorageFactory) pathFor(id ObjectID) string {
	prefix := id.Hash
	if len(prefix) > 2 {
		prefix = prefix[:2]
	}
	return filepath.Join(f.Root, id.Namespace, prefix, id.Hash+".sqlite")
}

type SQLiteStorage struct {
	db      *sql.DB
	path    string
	id      ObjectID
	factory *SQLiteStorageFactory
}

func (s *SQLiteStorage) init(ctx context.Context) error {
	stmts := []string{
		`PRAGMA journal_mode=WAL`,
		`CREATE TABLE IF NOT EXISTS kv (
			key TEXT PRIMARY KEY,
			value_json BLOB NOT NULL,
			updated_at_ms INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS meta (
			key TEXT PRIMARY KEY,
			value_json BLOB NOT NULL,
			updated_at_ms INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS alarms (
			singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
			due_at_ms INTEGER NOT NULL,
			updated_at_ms INTEGER NOT NULL
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return wrap(CodeStorageError, "initialize object sqlite schema", err)
		}
	}
	return nil
}

func (s *SQLiteStorage) Get(ctx context.Context, key string) (any, bool, error) {
	return storageGet(ctx, s.db, key)
}

func (s *SQLiteStorage) Put(ctx context.Context, key string, value any) error {
	return storagePut(ctx, s.db, key, value)
}

func (s *SQLiteStorage) Delete(ctx context.Context, key string) (bool, error) {
	return storageDelete(ctx, s.db, key)
}

func (s *SQLiteStorage) List(ctx context.Context, prefix string, limit int) (map[string]any, error) {
	return storageList(ctx, s.db, prefix, limit)
}

func (s *SQLiteStorage) Transaction(ctx context.Context, fn func(StorageTx) error) error {
	if fn == nil {
		return coded(CodeBadRequest, "storage transaction callback is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return wrap(CodeStorageError, "begin storage transaction", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if err := fn(sqliteStorageTx{tx: tx}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return wrap(CodeStorageError, "commit storage transaction", err)
	}
	committed = true
	return nil
}

func (s *SQLiteStorage) SetAlarm(ctx context.Context, dueAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO alarms (singleton, due_at_ms, updated_at_ms)
		VALUES (1, ?, ?)
		ON CONFLICT(singleton) DO UPDATE SET due_at_ms = excluded.due_at_ms, updated_at_ms = excluded.updated_at_ms`, dueAt.UnixMilli(), time.Now().UnixMilli())
	if err != nil {
		return wrap(CodeStorageError, "set object alarm", err)
	}
	if s.factory != nil {
		if err := s.factory.SetAlarmIndex(ctx, s.id, dueAt); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStorage) GetAlarm(ctx context.Context) (*time.Time, error) {
	var dueAtMS int64
	err := s.db.QueryRowContext(ctx, `SELECT due_at_ms FROM alarms WHERE singleton = 1`).Scan(&dueAtMS)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, wrap(CodeStorageError, "get object alarm", err)
	}
	dueAt := time.UnixMilli(dueAtMS)
	return &dueAt, nil
}

func (s *SQLiteStorage) DeleteAlarm(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM alarms WHERE singleton = 1`); err != nil {
		return wrap(CodeStorageError, "delete object alarm", err)
	}
	if s.factory != nil {
		if err := s.factory.DeleteAlarmIndex(ctx, s.id); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStorage) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

type sqliteStorageTx struct {
	tx *sql.Tx
}

func (t sqliteStorageTx) Get(ctx context.Context, key string) (any, bool, error) {
	return storageGet(ctx, t.tx, key)
}

func (t sqliteStorageTx) Put(ctx context.Context, key string, value any) error {
	return storagePut(ctx, t.tx, key, value)
}

func (t sqliteStorageTx) Delete(ctx context.Context, key string) (bool, error) {
	return storageDelete(ctx, t.tx, key)
}

func (t sqliteStorageTx) List(ctx context.Context, prefix string, limit int) (map[string]any, error) {
	return storageList(ctx, t.tx, prefix, limit)
}

type queryExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func storageGet(ctx context.Context, qe queryExecer, key string) (any, bool, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, false, coded(CodeBadRequest, "storage key is required")
	}
	var data []byte
	err := qe.QueryRowContext(ctx, `SELECT value_json FROM kv WHERE key = ?`, key).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, wrap(CodeStorageError, "get storage key", err)
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, false, wrap(CodeStorageError, "decode storage value", err)
	}
	return value, true, nil
}

func storagePut(ctx context.Context, qe queryExecer, key string, value any) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return coded(CodeBadRequest, "storage key is required")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return wrap(CodeBadRequest, "encode storage value", err)
	}
	_, err = qe.ExecContext(ctx, `INSERT INTO kv (key, value_json, updated_at_ms)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json, updated_at_ms = excluded.updated_at_ms`, key, data, time.Now().UnixMilli())
	if err != nil {
		return wrap(CodeStorageError, "put storage key", err)
	}
	return nil
}

func storageDelete(ctx context.Context, qe queryExecer, key string) (bool, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return false, coded(CodeBadRequest, "storage key is required")
	}
	result, err := qe.ExecContext(ctx, `DELETE FROM kv WHERE key = ?`, key)
	if err != nil {
		return false, wrap(CodeStorageError, "delete storage key", err)
	}
	rows, _ := result.RowsAffected()
	return rows > 0, nil
}

func storageList(ctx context.Context, qe queryExecer, prefix string, limit int) (map[string]any, error) {
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	like := prefix + "%"
	rows, err := qe.QueryContext(ctx, `SELECT key, value_json FROM kv WHERE key LIKE ? ORDER BY key LIMIT ?`, like, limit)
	if err != nil {
		return nil, wrap(CodeStorageError, "list storage keys", err)
	}
	defer rows.Close()

	ret := map[string]any{}
	for rows.Next() {
		var key string
		var data []byte
		if err := rows.Scan(&key, &data); err != nil {
			return nil, wrap(CodeStorageError, "scan storage row", err)
		}
		var value any
		if err := json.Unmarshal(data, &value); err != nil {
			return nil, wrap(CodeStorageError, fmt.Sprintf("decode storage value for %q", key), err)
		}
		ret[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, wrap(CodeStorageError, "iterate storage keys", err)
	}
	return ret, nil
}
