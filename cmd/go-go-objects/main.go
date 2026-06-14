package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-go-golems/go-go-objects/pkg/durableobjects"
	"gopkg.in/yaml.v3"
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
	bundlePath := flag.String("bundle", "", "Path to a CommonJS bundle exporting objects; defaults to built-in counter demo")
	manifestPath := flag.String("manifest", "", "Optional path to a JSON/YAML namespace manifest; defaults to deriving CAMEL_CASE namespaces from exports.objects")
	cpuTimeout := flag.Duration("cpu-timeout", 2*time.Second, "Per-dispatch JavaScript CPU timeout")
	idleTimeout := flag.Duration("idle-timeout", 5*time.Minute, "Idle actor eviction timeout")
	alarmInterval := flag.Duration("alarm-interval", time.Second, "Alarm scheduler interval; 0 disables background alarm dispatch")
	idleInterval := flag.Duration("idle-interval", time.Minute, "Idle eviction interval; 0 disables background idle eviction")
	devErrors := flag.Bool("dev-errors", true, "Return detailed gateway errors")
	flag.Parse()

	manifest, bundleSource, err := loadInputs(*bundlePath, *manifestPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load inputs: %v\n", err)
		os.Exit(1)
	}

	mgr, err := durableobjects.NewManager(
		manifest,
		durableobjects.NewBundle(bundleSource),
		durableobjects.NewSQLiteStorageFactory(*storageRoot),
		durableobjects.Options{CPUTimeout: *cpuTimeout, IdleTimeout: *idleTimeout},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create manager: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = mgr.Close(context.Background()) }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *alarmInterval > 0 {
		go func() {
			_ = durableobjects.NewAlarmScheduler(mgr, *alarmInterval, 100, func(err error) {
				fmt.Fprintf(os.Stderr, "alarm scheduler: %v\n", err)
			}).Run(ctx)
		}()
	}
	if *idleInterval > 0 {
		go func() {
			_ = durableobjects.NewIdleEvictor(mgr, *idleInterval, func(err error) {
				fmt.Fprintf(os.Stderr, "idle evictor: %v\n", err)
			}).Run(ctx)
		}()
	}

	server := &http.Server{
		Addr:              *addr,
		Handler:           durableobjects.NewGateway(mgr, durableobjects.GatewayOptions{DevErrors: *devErrors}),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	fmt.Printf("go-go-objects listening on http://%s\n", *addr)
	if *bundlePath == "" {
		fmt.Println("using built-in COUNTER demo")
		fmt.Println("try: curl -X POST http://" + *addr + "/rpc/COUNTER/global/increment -d '[1]'")
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		os.Exit(1)
	}
}

func loadInputs(bundlePath, manifestPath string) (durableobjects.Manifest, string, error) {
	if bundlePath == "" && manifestPath == "" {
		return durableobjects.Manifest{}, counterBundle, nil
	}
	if bundlePath == "" {
		return durableobjects.Manifest{}, "", fmt.Errorf("--manifest requires --bundle")
	}
	bundle, err := os.ReadFile(bundlePath)
	if err != nil {
		return durableobjects.Manifest{}, "", fmt.Errorf("read bundle %q: %w", bundlePath, err)
	}
	if manifestPath == "" {
		return durableobjects.Manifest{}, string(bundle), nil
	}
	manifest, err := loadManifestFile(manifestPath)
	if err != nil {
		return durableobjects.Manifest{}, "", err
	}
	return manifest, string(bundle), nil
}

func loadManifestFile(path string) (durableobjects.Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return durableobjects.Manifest{}, fmt.Errorf("read manifest %q: %w", path, err)
	}
	var manifest durableobjects.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		if yamlErr := yaml.Unmarshal(data, &manifest); yamlErr != nil {
			return durableobjects.Manifest{}, fmt.Errorf("decode manifest %q as JSON (%v) or YAML (%v)", path, err, yamlErr)
		}
	}
	if err := manifest.Validate(); err != nil {
		return durableobjects.Manifest{}, err
	}
	return manifest, nil
}
