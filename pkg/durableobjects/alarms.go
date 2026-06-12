package durableobjects

import (
	"context"
	"database/sql"
	"path/filepath"
	"time"
)

func (f *SQLiteStorageFactory) alarmIndexPath() string {
	return filepath.Join(f.Root, "alarms.sqlite")
}

func (f *SQLiteStorageFactory) openAlarmIndex(ctx context.Context) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", f.alarmIndexPath())
	if err != nil {
		return nil, wrap(CodeStorageError, "open alarm index", err)
	}
	stmts := []string{
		`PRAGMA journal_mode=WAL`,
		`CREATE TABLE IF NOT EXISTS object_alarms (
			object_hash TEXT PRIMARY KEY,
			namespace TEXT NOT NULL,
			name TEXT NOT NULL,
			due_at_ms INTEGER NOT NULL,
			updated_at_ms INTEGER NOT NULL
		)`,
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			_ = db.Close()
			return nil, wrap(CodeStorageError, "initialize alarm index", err)
		}
	}
	return db, nil
}

func (f *SQLiteStorageFactory) SetAlarmIndex(ctx context.Context, id ObjectID, dueAt time.Time) error {
	db, err := f.openAlarmIndex(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, `INSERT INTO object_alarms (object_hash, namespace, name, due_at_ms, updated_at_ms)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(object_hash) DO UPDATE SET namespace = excluded.namespace, name = excluded.name, due_at_ms = excluded.due_at_ms, updated_at_ms = excluded.updated_at_ms`,
		id.Hash, id.Namespace, id.Name, dueAt.UnixMilli(), time.Now().UnixMilli())
	if err != nil {
		return wrap(CodeStorageError, "set alarm index", err)
	}
	return nil
}

func (f *SQLiteStorageFactory) DeleteAlarmIndex(ctx context.Context, id ObjectID) error {
	db, err := f.openAlarmIndex(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `DELETE FROM object_alarms WHERE object_hash = ?`, id.Hash); err != nil {
		return wrap(CodeStorageError, "delete alarm index", err)
	}
	return nil
}

func (f *SQLiteStorageFactory) DueAlarms(ctx context.Context, now time.Time, limit int) ([]AlarmRecord, error) {
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	db, err := f.openAlarmIndex(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT namespace, name, object_hash, due_at_ms FROM object_alarms WHERE due_at_ms <= ? ORDER BY due_at_ms LIMIT ?`, now.UnixMilli(), limit)
	if err != nil {
		return nil, wrap(CodeStorageError, "query due alarms", err)
	}
	defer rows.Close()

	var records []AlarmRecord
	for rows.Next() {
		var namespace, name, hash string
		var dueAtMS int64
		if err := rows.Scan(&namespace, &name, &hash, &dueAtMS); err != nil {
			return nil, wrap(CodeStorageError, "scan due alarm", err)
		}
		records = append(records, AlarmRecord{ID: ObjectID{Namespace: namespace, Name: name, Hash: hash}, DueAt: time.UnixMilli(dueAtMS)})
	}
	if err := rows.Err(); err != nil {
		return nil, wrap(CodeStorageError, "iterate due alarms", err)
	}
	return records, nil
}

type AlarmIndexer interface {
	DueAlarms(ctx context.Context, now time.Time, limit int) ([]AlarmRecord, error)
	DeleteAlarmIndex(ctx context.Context, id ObjectID) error
}
