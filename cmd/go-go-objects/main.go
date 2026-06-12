package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-go-golems/go-go-objects/pkg/durableobjects"
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
  fetch(req) {
    if (req.path === "/count") {
      return { status: 200, body: String(this.state.storage.get("count") || 0) };
    }
    return { status: 404, body: "not found" };
  }
  alarm() {
    const current = this.state.storage.get("alarmCount") || 0;
    this.state.storage.put("alarmCount", current + 1);
  }
}
exports.objects = { Counter };
`

func main() {
	addr := flag.String("addr", "127.0.0.1:8787", "HTTP listen address")
	storageRoot := flag.String("storage", "./var/durable-objects", "SQLite storage root")
	flag.Parse()

	mgr, err := durableobjects.NewManager(
		durableobjects.Manifest{Objects: map[string]string{"COUNTER": "Counter"}},
		durableobjects.NewBundle(counterBundle),
		durableobjects.NewSQLiteStorageFactory(*storageRoot),
		durableobjects.Options{CPUTimeout: 2 * time.Second, IdleTimeout: 5 * time.Minute},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create manager: %v\n", err)
		os.Exit(1)
	}
	defer mgr.Close(context.Background())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		_ = durableobjects.NewAlarmScheduler(mgr, time.Second, 100, func(err error) {
			fmt.Fprintf(os.Stderr, "alarm scheduler: %v\n", err)
		}).Run(ctx)
	}()
	go func() {
		_ = durableobjects.NewIdleEvictor(mgr, time.Minute, func(err error) {
			fmt.Fprintf(os.Stderr, "idle evictor: %v\n", err)
		}).Run(ctx)
	}()

	server := &http.Server{
		Addr:              *addr,
		Handler:           durableobjects.NewGateway(mgr, durableobjects.GatewayOptions{DevErrors: true}),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	fmt.Printf("go-go-objects listening on http://%s\n", *addr)
	fmt.Println("try: curl -X POST http://" + *addr + "/rpc/COUNTER/global/increment -d '[1]'")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		os.Exit(1)
	}
}
