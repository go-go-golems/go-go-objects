package durableobjectsprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/go-go-golems/go-go-objects/pkg/durableobjects"
)

type serveSettings struct {
	Addr            string `glazed:"addr" json:"addr"`
	MaxRequestBytes int64  `glazed:"max-request-bytes" json:"maxRequestBytes"`
	DevErrors       bool   `glazed:"dev-errors" json:"devErrors"`
}

type serveCommand struct {
	*cmds.CommandDescription
	host          providerapi.HostServices
	baseConfig    settings
	serveDefaults serveSettings
	out           io.Writer
}

var _ cmds.BareCommand = (*serveCommand)(nil)

func newServeCommandSet(ctx providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
	baseConfig := defaultSettings()
	if len(ctx.Config) > 0 && string(ctx.Config) != "null" {
		if err := json.Unmarshal(ctx.Config, &baseConfig); err != nil {
			return nil, fmt.Errorf("decode durableobjects serve provider config: %w", err)
		}
	}
	sections, err := newCapability().GlazedConfigSections(providerapi.SectionRequest{CommandProviderID: ctx.Name, PackageID: ctx.PackageID})
	if err != nil {
		return nil, err
	}
	desc := cmds.NewCommandDescription(
		"serve",
		cmds.WithShort("Serve Durable Objects RPC/fetch HTTP endpoints"),
		cmds.WithLong("Start a Durable Objects gateway server backed by the configured bundle, storage root, alarm scheduler, and idle evictor."),
		cmds.WithFlags(
			fields.New("addr", fields.TypeString, fields.WithDefault("127.0.0.1:8787"), fields.WithHelp("HTTP listen address")),
			fields.New("max-request-bytes", fields.TypeInteger, fields.WithDefault(64<<20), fields.WithHelp("Maximum gateway request body size in bytes")),
			fields.New("dev-errors", fields.TypeBool, fields.WithDefault(true), fields.WithHelp("Return detailed gateway errors")),
		),
		cmds.WithSections(sections...),
	)
	return &providerapi.CommandSet{Commands: []cmds.Command{&serveCommand{
		CommandDescription: desc,
		host:               ctx.Host,
		baseConfig:         baseConfig,
		serveDefaults:      serveSettings{Addr: "127.0.0.1:8787", MaxRequestBytes: 64 << 20, DevErrors: true},
		out:                os.Stderr,
	}}}, nil
}

func (c *serveCommand) Run(ctx context.Context, vals *values.Values) error {
	cfg := c.baseConfig
	if vals != nil {
		public := defaultSettings()
		public.Enabled = false
		if err := vals.DecodeSectionInto("durableobjects", &public); err != nil {
			return err
		}
		if public.Enabled {
			mergePublicSettings(&cfg, public)
		}
	}
	serve := c.serveDefaults
	if vals != nil {
		if err := vals.DecodeSectionInto(schema.DefaultSlug, &serve); err != nil {
			return err
		}
	}
	return runServeCommand(ctx, c.host, cfg, serve, c.out)
}

func mergePublicSettings(dst *settings, src settings) {
	if strings.TrimSpace(src.StorageRoot) != "" {
		dst.StorageRoot = src.StorageRoot
	}
	if strings.TrimSpace(src.BundlePath) != "" {
		dst.BundlePath = src.BundlePath
		dst.BundleAsset = ""
	}
	if strings.TrimSpace(src.ManifestPath) != "" {
		dst.ManifestPath = src.ManifestPath
		dst.ManifestAsset = ""
	}
	if strings.TrimSpace(src.CPUTimeout) != "" {
		dst.CPUTimeout = src.CPUTimeout
	}
	if strings.TrimSpace(src.IdleTimeout) != "" {
		dst.IdleTimeout = src.IdleTimeout
	}
	if strings.TrimSpace(src.AlarmInterval) != "" {
		dst.AlarmInterval = src.AlarmInterval
	}
	if strings.TrimSpace(src.IdleInterval) != "" {
		dst.IdleInterval = src.IdleInterval
	}
}

func runServeCommand(ctx context.Context, host providerapi.HostServices, cfg settings, serve serveSettings, out io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if out == nil {
		out = io.Discard
	}
	addr := strings.TrimSpace(serve.Addr)
	if addr == "" {
		addr = "127.0.0.1:8787"
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	return serveOnListener(ctx, listener, host, cfg, serve, out)
}

func serveOnListener(ctx context.Context, listener net.Listener, host providerapi.HostServices, cfg settings, serve serveSettings, out io.Writer) error {
	if listener == nil {
		return fmt.Errorf("listener is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if out == nil {
		out = io.Discard
	}
	bundleSource, err := loadBundleSource(host, cfg)
	if err != nil {
		_ = listener.Close()
		return err
	}
	manifest, err := loadConfiguredManifest(host, cfg)
	if err != nil {
		_ = listener.Close()
		return err
	}
	cpuTimeout, err := parseDurationSetting("cpu-timeout", cfg.CPUTimeout)
	if err != nil {
		_ = listener.Close()
		return err
	}
	idleTimeout, err := parseDurationSetting("idle-timeout", cfg.IdleTimeout)
	if err != nil {
		_ = listener.Close()
		return err
	}
	alarmInterval, err := parseDurationSetting("alarm-interval", cfg.AlarmInterval)
	if err != nil {
		_ = listener.Close()
		return err
	}
	idleInterval, err := parseDurationSetting("idle-interval", cfg.IdleInterval)
	if err != nil {
		_ = listener.Close()
		return err
	}

	serveCtx, stopSignals := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	serverRuntime, err := durableobjects.NewServer(serveCtx, durableobjects.ServerOptions{
		Manifest:        manifest,
		BundleSource:    bundleSource,
		StorageRoot:     cfg.StorageRoot,
		CPUTimeout:      cpuTimeout,
		IdleTimeout:     idleTimeout,
		AlarmInterval:   alarmInterval,
		IdleInterval:    idleInterval,
		MaxRequestBytes: serve.MaxRequestBytes,
		DevErrors:       serve.DevErrors,
		ErrorHandler: func(err error) {
			_, _ = fmt.Fprintf(out, "durableobjects serve background error: %v\n", err)
		},
	})
	if err != nil {
		_ = listener.Close()
		return err
	}
	defer func() { _ = serverRuntime.Close(context.Background()) }()

	httpServer := &http.Server{Addr: listener.Addr().String(), Handler: serverRuntime.Handler, ReadHeaderTimeout: 5 * time.Second}
	serverErr := make(chan error, 1)
	go func() { serverErr <- httpServer.Serve(listener) }()
	_, _ = fmt.Fprintf(out, "durableobjects serve listening on http://%s\n", listener.Addr().String())
	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-serveCtx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}
