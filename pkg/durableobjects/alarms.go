package durableobjects

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (f *SQLiteStorageFactory) alarmIndexPath() (string, error) {
	if f == nil {
		return "", coded(CodeBadRequest, "sqlite storage root is required")
	}
	root, err := cleanStorageRoot(f.Root)
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, "alarms.sqlite")
	if err := ensurePathWithinRoot(root, path); err != nil {
		return "", err
	}
	return path, nil
}

func (f *SQLiteStorageFactory) openAlarmIndex(ctx context.Context) (*sql.DB, error) {
	path, err := f.alarmIndexPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, wrap(CodeStorageError, "create alarm index storage directory", err)
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, wrap(CodeStorageError, "open alarm index", err)
	}
	if err := ensureSQLiteSchemaVersion(ctx, db, "alarm index sqlite database"); err != nil {
		_ = db.Close()
		return nil, err
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
	defer func() { _ = db.Close() }()
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
	defer func() { _ = db.Close() }()
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
	defer func() { _ = db.Close() }()
	rows, err := db.QueryContext(ctx, `SELECT namespace, name, object_hash, due_at_ms FROM object_alarms WHERE due_at_ms <= ? ORDER BY due_at_ms LIMIT ?`, now.UnixMilli(), limit)
	if err != nil {
		return nil, wrap(CodeStorageError, "query due alarms", err)
	}
	defer func() { _ = rows.Close() }()

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

type AlarmReconcileResult struct {
	ScannedObjects int
	RepairedIndex  int
	RemovedStale   int
}

func (f *SQLiteStorageFactory) ReconcileAlarmIndex(ctx context.Context) (AlarmReconcileResult, error) {
	result := AlarmReconcileResult{}
	db, err := f.openAlarmIndex(ctx)
	if err != nil {
		return result, err
	}
	defer func() { _ = db.Close() }()
	local, err := f.localAlarms(ctx)
	if err != nil {
		return result, err
	}
	result.ScannedObjects = len(local)
	indexed, err := indexedAlarms(ctx, db)
	if err != nil {
		return result, err
	}
	for hash, alarm := range local {
		current, ok := indexed[hash]
		if ok && current.DueAt.Equal(alarm.DueAt) && current.ID.Namespace == alarm.ID.Namespace && current.ID.Name == alarm.ID.Name {
			continue
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO object_alarms (object_hash, namespace, name, due_at_ms, updated_at_ms)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(object_hash) DO UPDATE SET namespace = excluded.namespace, name = excluded.name, due_at_ms = excluded.due_at_ms, updated_at_ms = excluded.updated_at_ms`,
			alarm.ID.Hash, alarm.ID.Namespace, alarm.ID.Name, alarm.DueAt.UnixMilli(), time.Now().UnixMilli()); err != nil {
			return result, wrap(CodeStorageError, "repair alarm index", err)
		}
		result.RepairedIndex++
	}
	for hash := range indexed {
		if _, ok := local[hash]; ok {
			continue
		}
		if _, err := db.ExecContext(ctx, `DELETE FROM object_alarms WHERE object_hash = ?`, hash); err != nil {
			return result, wrap(CodeStorageError, "remove stale alarm index", err)
		}
		result.RemovedStale++
	}
	return result, nil
}

func (f *SQLiteStorageFactory) localAlarms(ctx context.Context) (map[string]AlarmRecord, error) {
	alarms := map[string]AlarmRecord{}
	if f == nil || strings.TrimSpace(f.Root) == "" {
		return alarms, nil
	}
	if _, err := os.Stat(f.Root); os.IsNotExist(err) {
		return alarms, nil
	}
	err := filepath.WalkDir(f.Root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Base(path) == "alarms.sqlite" || !strings.HasSuffix(path, ".sqlite") {
			return nil
		}
		record, ok, err := readLocalAlarm(ctx, path)
		if err != nil {
			return err
		}
		if ok {
			alarms[record.ID.Hash] = record
		}
		return nil
	})
	if err != nil {
		return nil, wrap(CodeStorageError, "scan local alarms", err)
	}
	return alarms, nil
}

func readLocalAlarm(ctx context.Context, path string) (AlarmRecord, bool, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return AlarmRecord{}, false, wrap(CodeStorageError, "open object database for alarm reconciliation", err)
	}
	defer func() { _ = db.Close() }()
	id, ok, err := readObjectMetadata(ctx, db)
	if err != nil || !ok {
		return AlarmRecord{}, false, err
	}
	var dueAtMS int64
	err = db.QueryRowContext(ctx, `SELECT due_at_ms FROM alarms WHERE singleton = 1`).Scan(&dueAtMS)
	if err == sql.ErrNoRows {
		return AlarmRecord{}, false, nil
	}
	if err != nil {
		return AlarmRecord{}, false, wrap(CodeStorageError, "read local alarm", err)
	}
	return AlarmRecord{ID: id, DueAt: time.UnixMilli(dueAtMS)}, true, nil
}

func readObjectMetadata(ctx context.Context, db *sql.DB) (ObjectID, bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT key, value_json FROM meta WHERE key IN ('object.namespace', 'object.name', 'object.hash')`)
	if err != nil {
		return ObjectID{}, false, nil
	}
	defer func() { _ = rows.Close() }()
	values := map[string]string{}
	for rows.Next() {
		var key string
		var data []byte
		if err := rows.Scan(&key, &data); err != nil {
			return ObjectID{}, false, wrap(CodeStorageError, "scan object metadata", err)
		}
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return ObjectID{}, false, wrap(CodeStorageError, "decode object metadata", err)
		}
		values[key] = value
	}
	if err := rows.Err(); err != nil {
		return ObjectID{}, false, wrap(CodeStorageError, "iterate object metadata", err)
	}
	id := ObjectID{Namespace: values["object.namespace"], Name: values["object.name"], Hash: values["object.hash"]}
	if id.IsZero() {
		return ObjectID{}, false, nil
	}
	return id, true, nil
}

func indexedAlarms(ctx context.Context, db *sql.DB) (map[string]AlarmRecord, error) {
	rows, err := db.QueryContext(ctx, `SELECT namespace, name, object_hash, due_at_ms FROM object_alarms`)
	if err != nil {
		return nil, wrap(CodeStorageError, "query indexed alarms", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]AlarmRecord{}
	for rows.Next() {
		var namespace, name, hash string
		var dueAtMS int64
		if err := rows.Scan(&namespace, &name, &hash, &dueAtMS); err != nil {
			return nil, wrap(CodeStorageError, "scan indexed alarm", err)
		}
		out[hash] = AlarmRecord{ID: ObjectID{Namespace: namespace, Name: name, Hash: hash}, DueAt: time.UnixMilli(dueAtMS)}
	}
	if err := rows.Err(); err != nil {
		return nil, wrap(CodeStorageError, "iterate indexed alarms", err)
	}
	return out, nil
}

type AlarmIndexer interface {
	DueAlarms(ctx context.Context, now time.Time, limit int) ([]AlarmRecord, error)
	DeleteAlarmIndex(ctx context.Context, id ObjectID) error
}

type AlarmReconciler interface {
	ReconcileAlarmIndex(ctx context.Context) (AlarmReconcileResult, error)
}
