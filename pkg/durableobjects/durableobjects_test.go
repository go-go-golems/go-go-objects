package durableobjects

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

const counterBundle = `
class Counter {
  constructor(state, env) {
    this.state = state;
    this.env = env;
  }
  increment(by) {
    const current = this.state.storage.get("count") || 0;
    const next = current + (by || 1);
    this.state.storage.put("count", next);
    return next;
  }
  value() {
    return this.state.storage.get("count") || 0;
  }
  fetch(req) {
    if (req.path === "/count") {
      return { status: 200, headers: { "X-Counter": "yes" }, body: String(this.state.storage.get("count") || 0) };
    }
    return { status: 404, body: "not found" };
  }
  scheduleAlarm(ms) {
    this.state.storage.setAlarm(Date.now() + ms);
    return this.state.storage.getAlarm();
  }
  alarmCount() {
    return this.state.storage.get("alarmCount") || 0;
  }
  spin(ms) {
    const end = Date.now() + ms;
    while (Date.now() < end) {}
    return true;
  }
  alarm() {
    const current = this.state.storage.get("alarmCount") || 0;
    this.state.storage.put("alarmCount", current + 1);
  }
}
exports.objects = { Counter };
`

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	mgr, err := NewManager(
		Manifest{Objects: map[string]string{"COUNTER": "Counter"}},
		NewBundle(counterBundle),
		NewSQLiteStorageFactory(t.TempDir()),
		Options{CPUTimeout: 2 * time.Second},
	)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close(context.Background()) })
	return mgr
}

func TestObjectIDRejectsUnsafeSegments(t *testing.T) {
	for _, tc := range []struct{ namespace, name string }{
		{"../COUNTER", "global"},
		{"COUNTER", "../global"},
		{"COUNTER/slash", "global"},
		{"COUNTER", "with/slash"},
		{"COUNTER", "bad\x00name"},
	} {
		if _, err := NewObjectID(tc.namespace, tc.name); err == nil {
			t.Fatalf("NewObjectID(%q, %q) succeeded, want error", tc.namespace, tc.name)
		}
	}
}

func TestObjectIDStable(t *testing.T) {
	id1, err := NewObjectID("COUNTER", "global")
	if err != nil {
		t.Fatalf("NewObjectID() error = %v", err)
	}
	id2, err := NewObjectID("COUNTER", "global")
	if err != nil {
		t.Fatalf("NewObjectID() error = %v", err)
	}
	if id1 != id2 {
		t.Fatalf("NewObjectID() not stable: %#v != %#v", id1, id2)
	}
	if id1.Hash == "" || id1.Namespace != "COUNTER" || id1.Name != "global" {
		t.Fatalf("unexpected id: %#v", id1)
	}
}

func TestExportNameToNamespace(t *testing.T) {
	tests := map[string]string{
		"Counter":       "COUNTER",
		"ChatRoom":      "CHAT_ROOM",
		"URLParser":     "URL_PARSER",
		"COUNTER":       "COUNTER",
		"Chat_Room":     "CHAT_ROOM",
		"chat-room":     "CHAT_ROOM",
		"Counter2DView": "COUNTER2_D_VIEW",
	}
	for input, want := range tests {
		if got := ExportNameToNamespace(input); got != want {
			t.Fatalf("ExportNameToNamespace(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestManagerDerivesManifestFromBundleExports(t *testing.T) {
	mgr, err := NewManager(Manifest{}, NewBundle(counterBundle), NewSQLiteStorageFactory(t.TempDir()), Options{CPUTimeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close(context.Background()) })
	id, err := NewObjectID("COUNTER", "global")
	if err != nil {
		t.Fatal(err)
	}
	if got := rpcNumber(t, mgr, id, "increment", []any{1}); got != 1 {
		t.Fatalf("increment = %v, want 1", got)
	}
}

func TestManagerDerivesCloudflareStyleNamespace(t *testing.T) {
	bundle := `
class ChatRoom { constructor(state, env) { this.state = state; this.env = env; } ping() { return "pong"; } }
exports.objects = { ChatRoom };
`
	mgr, err := NewManager(Manifest{}, NewBundle(bundle), NewSQLiteStorageFactory(t.TempDir()), Options{})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close(context.Background()) })
	id, err := NewObjectID("CHAT_ROOM", "general")
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal([]any{})
	result, err := mgr.Dispatch(context.Background(), Envelope{Kind: KindRPC, ID: id, Method: "ping", ArgsJSON: payload})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	var got string
	if err := json.Unmarshal(result.ValueJSON, &got); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if got != "pong" {
		t.Fatalf("ping = %q, want pong", got)
	}
}

func TestConcurrentFirstDispatchStartsOneActor(t *testing.T) {
	ctx := context.Background()
	var mu sync.Mutex
	starts := 0
	mgr, err := NewManager(
		Manifest{Objects: map[string]string{"COUNTER": "Counter"}},
		NewBundle(counterBundle),
		NewSQLiteStorageFactory(t.TempDir()),
		Options{CPUTimeout: 2 * time.Second, EventHook: func(event Event) {
			if event.Name == EventActorStart {
				mu.Lock()
				starts++
				mu.Unlock()
			}
		}},
	)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close(context.Background()) })
	id, err := NewObjectID("COUNTER", "concurrent")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errCh := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			payload, _ := json.Marshal([]any{1})
			_, err := mgr.Dispatch(ctx, Envelope{Kind: KindRPC, ID: id, Method: "increment", ArgsJSON: payload})
			errCh <- err
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("Dispatch() error = %v", err)
		}
	}
	mu.Lock()
	gotStarts := starts
	mu.Unlock()
	if gotStarts != 1 {
		t.Fatalf("actor starts = %d, want 1", gotStarts)
	}
	if got := rpcNumber(t, mgr, id, "value", nil); got != 20 {
		t.Fatalf("counter value = %v, want 20", got)
	}
}

func TestCounterRPCPersistsAcrossEviction(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	id, err := NewObjectID("COUNTER", "global")
	if err != nil {
		t.Fatal(err)
	}

	if got := rpcNumber(t, mgr, id, "increment", []any{1}); got != 1 {
		t.Fatalf("first increment = %v, want 1", got)
	}
	if err := mgr.Evict(ctx, id); err != nil {
		t.Fatalf("Evict() error = %v", err)
	}
	if got := rpcNumber(t, mgr, id, "increment", []any{1}); got != 2 {
		t.Fatalf("second increment after eviction = %v, want 2", got)
	}
}

func TestFetchGateway(t *testing.T) {
	mgr := newTestManager(t)
	id, err := NewObjectID("COUNTER", "global")
	if err != nil {
		t.Fatal(err)
	}
	_ = rpcNumber(t, mgr, id, "increment", []any{3})

	gateway := NewGateway(mgr, GatewayOptions{DevErrors: true})
	req := httptest.NewRequest(http.MethodGet, "/fetch/COUNTER/global/count", nil)
	w := httptest.NewRecorder()
	gateway.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got != "3" {
		t.Fatalf("body = %q, want 3", got)
	}
	if got := w.Header().Get("X-Counter"); got != "yes" {
		t.Fatalf("X-Counter = %q, want yes", got)
	}
}

func TestDispatchDueAlarmsCreatesStorageRootBeforeObjects(t *testing.T) {
	root := t.TempDir() + "/missing-root"
	mgr, err := NewManager(Manifest{Objects: map[string]string{"COUNTER": "Counter"}}, NewBundle(counterBundle), NewSQLiteStorageFactory(root), Options{CPUTimeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close(context.Background()) })
	dispatched, err := mgr.DispatchDueAlarms(context.Background(), time.Now(), 10)
	if err != nil {
		t.Fatalf("DispatchDueAlarms() error = %v", err)
	}
	if dispatched != 0 {
		t.Fatalf("dispatched = %d, want 0", dispatched)
	}
}

func TestAlarmDispatchWakesEvictedActor(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	id, err := NewObjectID("COUNTER", "global")
	if err != nil {
		t.Fatal(err)
	}
	_ = rpcNumber(t, mgr, id, "scheduleAlarm", []any{-1})
	if err := mgr.Evict(ctx, id); err != nil {
		t.Fatalf("Evict() error = %v", err)
	}
	dispatched, err := mgr.DispatchDueAlarms(ctx, time.Now(), 10)
	if err != nil {
		t.Fatalf("DispatchDueAlarms() error = %v", err)
	}
	if dispatched != 1 {
		t.Fatalf("dispatched = %d, want 1", dispatched)
	}
	if got := rpcNumber(t, mgr, id, "alarmCount", nil); got != 1 {
		t.Fatalf("alarmCount = %v, want 1", got)
	}
	dispatched, err = mgr.DispatchDueAlarms(ctx, time.Now(), 10)
	if err != nil {
		t.Fatalf("DispatchDueAlarms() second error = %v", err)
	}
	if dispatched != 0 {
		t.Fatalf("second dispatched = %d, want 0", dispatched)
	}
}

func TestDispatchDueAlarmsReconcilesMissingIndex(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	factory := NewSQLiteStorageFactory(root)
	mgr, err := NewManager(
		Manifest{Objects: map[string]string{"COUNTER": "Counter"}},
		NewBundle(counterBundle),
		factory,
		Options{CPUTimeout: 2 * time.Second},
	)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	id, err := NewObjectID("COUNTER", "reconcile")
	if err != nil {
		t.Fatal(err)
	}
	_ = rpcNumber(t, mgr, id, "scheduleAlarm", []any{-1})
	if err := factory.DeleteAlarmIndex(ctx, id); err != nil {
		t.Fatalf("DeleteAlarmIndex() error = %v", err)
	}
	if err := mgr.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	mgr, err = NewManager(
		Manifest{Objects: map[string]string{"COUNTER": "Counter"}},
		NewBundle(counterBundle),
		factory,
		Options{CPUTimeout: 2 * time.Second},
	)
	if err != nil {
		t.Fatalf("NewManager() after restart error = %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close(context.Background()) })
	dispatched, err := mgr.DispatchDueAlarms(ctx, time.Now(), 10)
	if err != nil {
		t.Fatalf("DispatchDueAlarms() error = %v", err)
	}
	if dispatched != 1 {
		t.Fatalf("dispatched = %d, want 1", dispatched)
	}
	if got := rpcNumber(t, mgr, id, "alarmCount", nil); got != 1 {
		t.Fatalf("alarmCount = %v, want 1", got)
	}
}

func TestEvictIdle(t *testing.T) {
	ctx := context.Background()
	mgr, err := NewManager(
		Manifest{Objects: map[string]string{"COUNTER": "Counter"}},
		NewBundle(counterBundle),
		NewSQLiteStorageFactory(t.TempDir()),
		Options{CPUTimeout: 2 * time.Second, IdleTimeout: time.Nanosecond},
	)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close(context.Background()) })
	id, err := NewObjectID("COUNTER", "global")
	if err != nil {
		t.Fatal(err)
	}
	_ = rpcNumber(t, mgr, id, "increment", []any{1})
	evicted, err := mgr.EvictIdle(ctx, time.Now().Add(time.Second))
	if err != nil {
		t.Fatalf("EvictIdle() error = %v", err)
	}
	if evicted != 1 {
		t.Fatalf("evicted = %d, want 1", evicted)
	}
	if got := rpcNumber(t, mgr, id, "value", nil); got != 1 {
		t.Fatalf("value after idle eviction = %v, want 1", got)
	}
}

func TestAlarmSchedulerTick(t *testing.T) {
	ctx := context.Background()
	mgr := newTestManager(t)
	id, err := NewObjectID("COUNTER", "scheduled")
	if err != nil {
		t.Fatal(err)
	}
	_ = rpcNumber(t, mgr, id, "scheduleAlarm", []any{-1})
	if err := mgr.Evict(ctx, id); err != nil {
		t.Fatalf("Evict() error = %v", err)
	}
	scheduler := NewAlarmScheduler(mgr, time.Second, 10, nil)
	dispatched, err := scheduler.Tick(ctx, time.Now())
	if err != nil {
		t.Fatalf("AlarmScheduler.Tick() error = %v", err)
	}
	if dispatched != 1 {
		t.Fatalf("dispatched = %d, want 1", dispatched)
	}
	if got := rpcNumber(t, mgr, id, "alarmCount", nil); got != 1 {
		t.Fatalf("alarmCount = %v, want 1", got)
	}
}

func TestIdleEvictorTick(t *testing.T) {
	ctx := context.Background()
	mgr, err := NewManager(
		Manifest{Objects: map[string]string{"COUNTER": "Counter"}},
		NewBundle(counterBundle),
		NewSQLiteStorageFactory(t.TempDir()),
		Options{CPUTimeout: 2 * time.Second, IdleTimeout: time.Nanosecond},
	)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close(context.Background()) })
	id, err := NewObjectID("COUNTER", "evictor")
	if err != nil {
		t.Fatal(err)
	}
	_ = rpcNumber(t, mgr, id, "increment", []any{1})
	evictor := NewIdleEvictor(mgr, time.Second, nil)
	evicted, err := evictor.Tick(ctx, time.Now().Add(time.Second))
	if err != nil {
		t.Fatalf("IdleEvictor.Tick() error = %v", err)
	}
	if evicted != 1 {
		t.Fatalf("evicted = %d, want 1", evicted)
	}
	if mgr.ActiveCount() != 0 {
		t.Fatalf("ActiveCount() = %d, want 0", mgr.ActiveCount())
	}
}

func TestEvictIdleDoesNotEvictActiveActor(t *testing.T) {
	ctx := context.Background()
	mgr, err := NewManager(
		Manifest{Objects: map[string]string{"COUNTER": "Counter"}},
		NewBundle(counterBundle),
		NewSQLiteStorageFactory(t.TempDir()),
		Options{CPUTimeout: time.Second, IdleTimeout: time.Nanosecond},
	)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close(context.Background()) })
	id, err := NewObjectID("COUNTER", "active")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		payload, _ := json.Marshal([]any{50})
		_, err := mgr.Dispatch(ctx, Envelope{Kind: KindRPC, ID: id, Method: "spin", ArgsJSON: payload})
		done <- err
	}()
	deadline := time.Now().Add(time.Second)
	for mgr.ActiveCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	evicted, err := mgr.EvictIdle(ctx, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("EvictIdle() error = %v", err)
	}
	if evicted != 0 {
		t.Fatalf("evicted active actor = %d, want 0", evicted)
	}
	if err := <-done; err != nil {
		t.Fatalf("active dispatch error = %v", err)
	}
}

func TestRPCGateway(t *testing.T) {
	gateway := NewGateway(newTestManager(t), GatewayOptions{DevErrors: true})
	req := httptest.NewRequest(http.MethodPost, "/rpc/COUNTER/global/increment", bytes.NewBufferString(`[2]`))
	w := httptest.NewRecorder()
	gateway.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var payload struct {
		OK     bool    `json:"ok"`
		Result float64 `json:"result"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, w.Body.String())
	}
	if !payload.OK || payload.Result != 2 {
		t.Fatalf("payload = %#v, want ok result 2", payload)
	}
}

func rpcNumber(t *testing.T, mgr *Manager, id ObjectID, method string, args []any) float64 {
	t.Helper()
	payload, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	result, err := mgr.Dispatch(context.Background(), Envelope{Kind: KindRPC, ID: id, Method: method, ArgsJSON: payload})
	if err != nil {
		t.Fatalf("Dispatch(%s) error = %v", method, err)
	}
	var got float64
	if err := json.Unmarshal(result.ValueJSON, &got); err != nil {
		t.Fatalf("decode result %q: %v", string(result.ValueJSON), err)
	}
	return got
}
