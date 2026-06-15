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
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
	"github.com/go-go-golems/go-go-objects/pkg/xgoja/providers/durableobjects/doc"
)

// newDocsCommandSet is the CommandSetProvider factory for the "docs" verb. It
// loads the embedded Glazed help pages into a fresh *help.HelpSystem and
// returns list/show/serve commands that read from it.
//
// The verb is self-contained: it builds its own HelpSystem from the same
// embedded filesystem that the HelpSource ships, so it works even in binaries
// that do not select the global help source. Both surfaces read identical
// bytes and therefore cannot diverge within one binary.
func newDocsCommandSet(_ providerapi.CommandSetContext) (*providerapi.CommandSet, error) {
	helpSystem := help.NewHelpSystem()
	if err := doc.AddDocToHelpSystem(helpSystem); err != nil {
		return nil, fmt.Errorf("load durableobjects docs: %w", err)
	}
	commands := []cmds.Command{
		newDocsListCommand(helpSystem),
		newDocsShowCommand(helpSystem),
		newDocsServeCommand(helpSystem),
	}
	return &providerapi.CommandSet{Commands: commands}, nil
}

// --- list -----------------------------------------------------------------

type docsListCommand struct {
	*cmds.CommandDescription
	hs *help.HelpSystem
}

var _ cmds.GlazeCommand = (*docsListCommand)(nil)

func newDocsListCommand(hs *help.HelpSystem) cmds.Command {
	return &docsListCommand{
		CommandDescription: cmds.NewCommandDescription(
			"list",
			cmds.WithParents("docs"),
			cmds.WithShort("List bundled go-go-objects help pages"),
			cmds.WithLong("List the Glazed help pages embedded by the durableobjects provider. Output is a Glaze table and can be rendered as text, JSON, CSV, etc. through standard Glazed flags."),
		),
		hs: hs,
	}
}

func (c *docsListCommand) RunIntoGlazeProcessor(ctx context.Context, _ *values.Values, gp middlewares.Processor) error {
	sections, err := c.hs.QuerySections("")
	if err != nil {
		return fmt.Errorf("query help sections: %w", err)
	}
	sort.Slice(sections, func(i, j int) bool {
		if sections[i] == nil || sections[j] == nil {
			return false
		}
		return sections[i].Slug < sections[j].Slug
	})
	for _, section := range sections {
		if section == nil {
			continue
		}
		if err := gp.AddRow(ctx, types.NewRow(
			types.MRP("slug", section.Slug),
			types.MRP("title", section.Title),
			types.MRP("sectionType", section.SectionType.String()),
			types.MRP("short", section.Short),
			types.MRP("topics", strings.Join(section.Topics, ",")),
		)); err != nil {
			return err
		}
	}
	return nil
}

// --- show -----------------------------------------------------------------

type docsShowCommand struct {
	*cmds.CommandDescription
	hs *help.HelpSystem
}

var _ cmds.WriterCommand = (*docsShowCommand)(nil)

type docsShowSettings struct {
	Slug string `glazed:"slug"`
}

func newDocsShowCommand(hs *help.HelpSystem) cmds.Command {
	return &docsShowCommand{
		CommandDescription: cmds.NewCommandDescription(
			"show",
			cmds.WithParents("docs"),
			cmds.WithShort("Print a bundled go-go-objects help page"),
			cmds.WithLong("Print the rendered Markdown body of one embedded help page, looked up by its Glazed slug (for example go-go-objects-js-api)."),
			cmds.WithArguments(
				fields.New("slug", fields.TypeString,
					fields.WithRequired(true),
					fields.WithHelp("Help page slug, e.g. go-go-objects-js-api")),
			),
		),
		hs: hs,
	}
}

func (c *docsShowCommand) RunIntoWriter(_ context.Context, vals *values.Values, w io.Writer) error {
	var settings docsShowSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	slug := strings.TrimSpace(settings.Slug)
	if slug == "" {
		return fmt.Errorf("slug is required")
	}
	section, err := c.hs.GetSectionWithSlug(slug)
	if err != nil {
		return fmt.Errorf("no help page with slug %q: %w", slug, err)
	}
	if section == nil {
		return fmt.Errorf("no help page with slug %q", slug)
	}
	fmt.Fprintf(w, "# %s\n\n%s\n", section.Title, section.Content)
	return nil
}

// --- serve ----------------------------------------------------------------

type docsServeCommand struct {
	*cmds.CommandDescription
	hs *help.HelpSystem
}

var _ cmds.BareCommand = (*docsServeCommand)(nil)

type docsServeSettings struct {
	Addr string `glazed:"addr"`
}

func newDocsServeCommand(hs *help.HelpSystem) cmds.Command {
	return &docsServeCommand{
		CommandDescription: cmds.NewCommandDescription(
			"serve",
			cmds.WithParents("docs"),
			cmds.WithShort("Serve bundled go-go-objects docs over HTTP"),
			cmds.WithLong("Start an HTTP server that exposes the embedded help pages as JSON. GET /docs lists pages; GET /docs/{slug} returns one page. Useful for non-CLI consumers and integration tests."),
			cmds.WithFlags(
				fields.New("addr", fields.TypeString, fields.WithDefault("127.0.0.1:8788"), fields.WithHelp("HTTP listen address")),
			),
		),
		hs: hs,
	}
}

func (c *docsServeCommand) Run(ctx context.Context, vals *values.Values) error {
	var settings docsServeSettings
	if err := vals.DecodeSectionInto(schema.DefaultSlug, &settings); err != nil {
		return err
	}
	addr := strings.TrimSpace(settings.Addr)
	if addr == "" {
		addr = "127.0.0.1:8788"
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	return serveDocsOnListener(ctx, listener, c.hs)
}

func serveDocsOnListener(ctx context.Context, listener net.Listener, hs *help.HelpSystem) error {
	if listener == nil {
		return fmt.Errorf("listener is required")
	}
	if hs == nil {
		_ = listener.Close()
		return fmt.Errorf("help system is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	mux := http.NewServeMux()
	registerDocsHTTPHandlers(mux, hs)

	serveCtx, stopSignals := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	httpServer := &http.Server{
		Addr:              listener.Addr().String(),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	serverErr := make(chan error, 1)
	go func() { serverErr <- httpServer.Serve(listener) }()
	fmt.Printf("durableobjects docs serve listening on http://%s\n", listener.Addr().String())
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

// docListItem is the JSON shape returned by GET /docs.
type docListItem struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	SectionType string `json:"sectionType"`
	Short       string `json:"summary"`
}

// docDetail is the JSON shape returned by GET /docs/{slug}.
type docDetail struct {
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	SectionType string   `json:"sectionType"`
	Short       string   `json:"summary"`
	Body        string   `json:"body"`
	Topics      []string `json:"topics"`
	Commands    []string `json:"commands"`
	Flags       []string `json:"flags"`
}

func registerDocsHTTPHandlers(mux *http.ServeMux, hs *help.HelpSystem) {
	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, r *http.Request) {
		sections, err := hs.QuerySections("")
		if err != nil {
			writeDocsHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		sort.Slice(sections, func(i, j int) bool {
			if sections[i] == nil || sections[j] == nil {
				return false
			}
			return sections[i].Slug < sections[j].Slug
		})
		items := make([]docListItem, 0, len(sections))
		for _, section := range sections {
			if section == nil {
				continue
			}
			items = append(items, docListItem{
				Slug:        section.Slug,
				Title:       section.Title,
				SectionType: section.SectionType.String(),
				Short:       section.Short,
			})
		}
		writeDocsHTTPJSON(w, http.StatusOK, items)
	})
	mux.HandleFunc("GET /docs/{slug}", func(w http.ResponseWriter, r *http.Request) {
		slug := strings.TrimPrefix(strings.TrimSpace(r.PathValue("slug")), "/")
		if slug == "" {
			writeDocsHTTPError(w, http.StatusBadRequest, fmt.Errorf("slug is required"))
			return
		}
		section, err := hs.GetSectionWithSlug(slug)
		if err != nil || section == nil {
			writeDocsHTTPError(w, http.StatusNotFound, fmt.Errorf("no help page with slug %q", slug))
			return
		}
		writeDocsHTTPJSON(w, http.StatusOK, docDetail{
			Slug:        section.Slug,
			Title:       section.Title,
			SectionType: section.SectionType.String(),
			Short:       section.Short,
			Body:        section.Content,
			Topics:      append([]string(nil), section.Topics...),
			Commands:    append([]string(nil), section.Commands...),
			Flags:       append([]string(nil), section.Flags...),
		})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeDocsHTTPError(w, http.StatusNotFound, fmt.Errorf("use GET /docs or GET /docs/{slug}"))
	})
}

func writeDocsHTTPJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeDocsHTTPError(w http.ResponseWriter, status int, err error) {
	writeDocsHTTPJSON(w, status, map[string]string{"error": err.Error()})
}
