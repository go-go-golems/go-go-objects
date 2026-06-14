package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/go-go-golems/go-go-objects/cmd/go-go-objects/doc"
	"github.com/go-go-golems/go-go-objects/pkg/durableobjects"
	"github.com/spf13/cobra"
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

type serveSettings struct {
	Addr          string `glazed:"addr"`
	StorageRoot   string `glazed:"storage"`
	BundlePath    string `glazed:"bundle"`
	ManifestPath  string `glazed:"manifest"`
	CPUTimeout    string `glazed:"cpu-timeout"`
	IdleTimeout   string `glazed:"idle-timeout"`
	AlarmInterval string `glazed:"alarm-interval"`
	IdleInterval  string `glazed:"idle-interval"`
	DevErrors     bool   `glazed:"dev-errors"`
}

type serveCommand struct {
	*cmds.CommandDescription
}

var _ cmds.BareCommand = (*serveCommand)(nil)

func newServeCommand() (*serveCommand, error) {
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}
	desc := cmds.NewCommandDescription(
		"go-go-objects",
		cmds.WithShort("Run Durable Objects-style JavaScript actors on goja"),
		cmds.WithLong(`Run a Durable Objects HTTP gateway backed by goja actors and SQLite storage.

Examples:
  go-go-objects --addr 127.0.0.1:8787 --storage ./var/durable-objects
  go-go-objects --bundle ./objects.js --manifest ./durableobjects.yaml
  curl -X POST http://127.0.0.1:8787/rpc/COUNTER/global/increment -d '[1]'

Use "go-go-objects help go-go-objects-js-api" for the JavaScript object API.`),
		cmds.WithFlags(
			fields.New("addr", fields.TypeString, fields.WithDefault("127.0.0.1:8787"), fields.WithHelp("HTTP listen address")),
			fields.New("storage", fields.TypeString, fields.WithDefault("./var/durable-objects"), fields.WithHelp("SQLite storage root")),
			fields.New("bundle", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Path to a CommonJS bundle exporting objects; defaults to built-in counter demo")),
			fields.New("manifest", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Optional path to a JSON/YAML namespace manifest; defaults to deriving CAMEL_CASE namespaces from exports.objects")),
			fields.New("cpu-timeout", fields.TypeString, fields.WithDefault("2s"), fields.WithHelp("Per-dispatch JavaScript CPU/Promise settlement timeout")),
			fields.New("idle-timeout", fields.TypeString, fields.WithDefault("5m"), fields.WithHelp("Idle actor eviction timeout")),
			fields.New("alarm-interval", fields.TypeString, fields.WithDefault("1s"), fields.WithHelp("Alarm scheduler interval; 0 disables background alarm dispatch")),
			fields.New("idle-interval", fields.TypeString, fields.WithDefault("1m"), fields.WithHelp("Idle eviction interval; 0 disables background idle eviction")),
			fields.New("dev-errors", fields.TypeBool, fields.WithDefault(true), fields.WithHelp("Return detailed gateway errors")),
		),
		cmds.WithSections(commandSettingsSection),
	)
	return &serveCommand{CommandDescription: desc}, nil
}

func (c *serveCommand) Run(ctx context.Context, vals *values.Values) error {
	var settings serveSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	return runServer(ctx, settings)
}

func main() {
	root, err := newRootCommand()
	if err != nil {
		fmt.Fprintf(os.Stderr, "create command: %v\n", err)
		os.Exit(1)
	}
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func newRootCommand() (*cobra.Command, error) {
	serve, err := newServeCommand()
	if err != nil {
		return nil, err
	}
	root, err := cli.BuildCobraCommandFromCommand(serve,
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpSections: []string{schema.DefaultSlug},
			MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
		}),
	)
	if err != nil {
		return nil, err
	}
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return logging.InitLoggerFromCobra(cmd)
	}
	if err := logging.AddLoggingSectionToRootCommand(root, "go-go-objects"); err != nil {
		return nil, err
	}
	setDefaultFlagValue(root, "log-level", "error")
	setDefaultFlagValue(root, "log-format", "text")

	helpSystem := help.NewHelpSystem()
	if err := doc.AddDocToHelpSystem(helpSystem); err != nil {
		return nil, err
	}
	help_cmd.SetupCobraRootCommand(helpSystem, root)
	return root, nil
}

func runServer(ctx context.Context, settings serveSettings) error {
	if ctx == nil {
		ctx = context.Background()
	}
	cpuTimeout, err := parseDurationFlag("cpu-timeout", settings.CPUTimeout)
	if err != nil {
		return err
	}
	idleTimeout, err := parseDurationFlag("idle-timeout", settings.IdleTimeout)
	if err != nil {
		return err
	}
	alarmInterval, err := parseDurationFlag("alarm-interval", settings.AlarmInterval)
	if err != nil {
		return err
	}
	idleInterval, err := parseDurationFlag("idle-interval", settings.IdleInterval)
	if err != nil {
		return err
	}
	manifest, bundleSource, err := loadInputs(settings.BundlePath, settings.ManifestPath)
	if err != nil {
		return fmt.Errorf("load inputs: %w", err)
	}

	mgr, err := durableobjects.NewManager(
		manifest,
		durableobjects.NewBundle(bundleSource),
		durableobjects.NewSQLiteStorageFactory(settings.StorageRoot),
		durableobjects.Options{CPUTimeout: cpuTimeout, IdleTimeout: idleTimeout},
	)
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}
	defer func() { _ = mgr.Close(context.Background()) }()

	serverCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if alarmInterval > 0 {
		go func() {
			_ = durableobjects.NewAlarmScheduler(mgr, alarmInterval, 100, func(err error) {
				fmt.Fprintf(os.Stderr, "alarm scheduler: %v\n", err)
			}).Run(serverCtx)
		}()
	}
	if idleInterval > 0 {
		go func() {
			_ = durableobjects.NewIdleEvictor(mgr, idleInterval, func(err error) {
				fmt.Fprintf(os.Stderr, "idle evictor: %v\n", err)
			}).Run(serverCtx)
		}()
	}

	server := &http.Server{
		Addr:              settings.Addr,
		Handler:           durableobjects.NewGateway(mgr, durableobjects.GatewayOptions{DevErrors: settings.DevErrors}),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-serverCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	fmt.Printf("go-go-objects listening on http://%s\n", settings.Addr)
	if settings.BundlePath == "" {
		fmt.Println("using built-in COUNTER demo")
		fmt.Println("try: curl -X POST http://" + settings.Addr + "/rpc/COUNTER/global/increment -d '[1]'")
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

func parseDurationFlag(name, value string) (time.Duration, error) {
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse --%s duration %q: %w", name, value, err)
	}
	return d, nil
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

func setDefaultFlagValue(root *cobra.Command, name string, value string) {
	flag := root.PersistentFlags().Lookup(name)
	if flag == nil {
		return
	}
	flag.DefValue = value
	_ = flag.Value.Set(value)
}
