package durableobjects

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
