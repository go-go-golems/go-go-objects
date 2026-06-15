package durableobjectsprovider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

func newDocsCommandSetForTest(t *testing.T) (*providerapi.CommandSet, *help.HelpSystem) {
	t.Helper()
	hs := help.NewHelpSystem()
	if err := doc.AddDocToHelpSystem(hs); err != nil {
		t.Fatalf("load docs: %v", err)
	}
	set, err := newDocsCommandSet(providerapi.CommandSetContext{})
	if err != nil {
		t.Fatalf("newDocsCommandSet: %v", err)
	}
	return set, hs
}

// TestDocsCommandSetHasThreeCommands asserts the factory returns exactly the
// list/show/serve commands, and that the provider exposes them by name.
func TestDocsCommandSetHasThreeCommands(t *testing.T) {
	set, _ := newDocsCommandSetForTest(t)
	names := make([]string, 0, len(set.Commands))
	for _, c := range set.Commands {
		names = append(names, c.Description().Name)
	}
	want := []string{"list", "show", "serve"}
	for _, w := range want {
		found := false
		for _, n := range names {
			if n == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected command %q in docs set, got %v", w, names)
		}
	}
	if len(set.Commands) != len(want) {
		t.Errorf("expected %d commands, got %d (%v)", len(want), len(set.Commands), names)
	}
}

// TestRegisterExposesDocsCommandProvider asserts the "docs" command set is
// resolvable after Register, mirroring how the app layer discovers verbs.
func TestRegisterExposesDocsCommandProvider(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("Register: %v", err)
	}
	csp, ok := registry.ResolveCommandSetProvider(PackageID, "docs")
	if !ok {
		t.Fatalf("ResolveCommandSetProvider(%q, %q) not found", PackageID, "docs")
	}
	if csp.Name != "docs" {
		t.Errorf("command set name = %q, want %q", csp.Name, "docs")
	}
	if csp.DefaultMount != "durableobjects" {
		t.Errorf("default mount = %q, want %q", csp.DefaultMount, "durableobjects")
	}
	set, err := csp.NewCommandSet(providerapi.CommandSetContext{})
	if err != nil {
		t.Fatalf("NewCommandSet: %v", err)
	}
	if len(set.Commands) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(set.Commands))
	}
}

// TestDocsListCommandEmitsAllPages asserts the list command emits one row per
// embedded help page with the expected slugs.
func TestDocsListCommandEmitsAllPages(t *testing.T) {
	set, _ := newDocsCommandSetForTest(t)
	list := findDocsCommand(t, set, "list")
	glaze, ok := list.(cmds.GlazeCommand)
	if !ok {
		t.Fatalf("list command is not a GlazeCommand: %T", list)
	}
	processor := middlewares.NewTableProcessor(middlewares.WithTableMiddleware(collectTableMiddleware{}))
	if err := glaze.RunIntoGlazeProcessor(context.Background(), values.New(), processor); err != nil {
		t.Fatalf("RunIntoGlazeProcessor: %v", err)
	}
	rows := processor.Table.Rows
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	gotSlugs := map[string]bool{}
	for _, row := range rows {
		if v, ok := row.Get("slug"); ok {
			gotSlugs[toString(v)] = true
		}
	}
	for _, want := range []string{"go-go-objects-overview", "go-go-objects-js-api", "go-go-objects-xgoja-provider"} {
		if !gotSlugs[want] {
			t.Errorf("missing slug %q in list output: %v", want, gotSlugs)
		}
	}
}

// TestDocsShowCommandPrintsBody asserts show renders the page body for a known
// slug and errors for an unknown one.
func TestDocsShowCommandPrintsBody(t *testing.T) {
	set, _ := newDocsCommandSetForTest(t)
	show := findDocsCommand(t, set, "show")
	writer, ok := show.(cmds.WriterCommand)
	if !ok {
		t.Fatalf("show command is not a WriterCommand: %T", show)
	}

	var buf strings.Builder
	vals := defaultSectionValues(t, map[string]any{"slug": "go-go-objects-overview"})
	if err := writer.RunIntoWriter(context.Background(), vals, &buf); err != nil {
		t.Fatalf("RunIntoWriter: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "go-go-objects") {
		t.Errorf("show output missing expected content, got: %q", out)
	}
	if !strings.Contains(out, "# ") {
		t.Errorf("show output missing title marker, got: %q", out)
	}

	// Unknown slug must error.
	var buf2 strings.Builder
	badVals := defaultSectionValues(t, map[string]any{"slug": "does-not-exist"})
	err := writer.RunIntoWriter(context.Background(), badVals, &buf2)
	if err == nil {
		t.Errorf("expected error for unknown slug, got output %q", buf2.String())
	}
}

// TestDocsServeHTTP asserts the serve command's HTTP surface returns the
// expected JSON for the list and detail endpoints.
func TestDocsServeHTTP(t *testing.T) {
	set, hs := newDocsCommandSetForTest(t)
	_ = set
	mux := http.NewServeMux()
	registerDocsHTTPHandlers(mux, hs)
	server := httptest.NewServer(mux)
	defer server.Close()

	// GET /docs -> array with all three slugs.
	resp, err := http.Get(server.URL + "/docs")
	if err != nil {
		t.Fatalf("GET /docs: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /docs status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var items []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("decode /docs: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("GET /docs returned %d items, want 3", len(items))
	}

	// GET /docs/{slug} -> detail with a body.
	resp2, err := http.Get(server.URL + "/docs/go-go-objects-js-api")
	if err != nil {
		t.Fatalf("GET /docs/{slug}: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("GET /docs/{slug} status = %d, want %d", resp2.StatusCode, http.StatusOK)
	}
	var detail map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	body, _ := detail["body"].(string)
	if body == "" {
		t.Errorf("detail body is empty; detail=%v", detail)
	}
	if detail["slug"] != "go-go-objects-js-api" {
		t.Errorf("detail slug = %v, want go-go-objects-js-api", detail["slug"])
	}

	// GET /docs/unknown -> 404.
	resp3, err := http.Get(server.URL + "/docs/unknown-slug")
	if err != nil {
		t.Fatalf("GET unknown: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusNotFound {
		t.Errorf("unknown slug status = %d, want %d", resp3.StatusCode, http.StatusNotFound)
	}
}

// findDocsCommand looks up a command by name in the set.
func findDocsCommand(t *testing.T, set *providerapi.CommandSet, name string) cmds.Command {
	t.Helper()
	for _, c := range set.Commands {
		if c.Description().Name == name {
			return c
		}
	}
	t.Fatalf("command %q not found in set", name)
	return nil
}

// defaultSectionValues builds a *values.Values carrying the given field values
// in the default section, so argument/flag-based commands can decode them.
func defaultSectionValues(t *testing.T, fieldValues map[string]any) *values.Values {
	t.Helper()
	section, err := schema.NewSection(
		schema.DefaultSlug,
		"docs test",
		schema.WithFields(buildFieldDefinitions(fieldValues)...),
	)
	if err != nil {
		t.Fatalf("NewSection: %v", err)
	}
	options := make([]values.SectionValuesOption, 0, len(fieldValues))
	for k, v := range fieldValues {
		options = append(options, values.WithFieldValue(k, v))
	}
	sectionValues, err := values.NewSectionValues(section, options...)
	if err != nil {
		t.Fatalf("NewSectionValues: %v", err)
	}
	return values.New(values.WithSectionValues(schema.DefaultSlug, sectionValues))
}

func buildFieldDefinitions(fieldValues map[string]any) []*fields.Definition {
	out := make([]*fields.Definition, 0, len(fieldValues))
	for k := range fieldValues {
		out = append(out, fields.New(k, fields.TypeString))
	}
	return out
}

func toString(v any) string {
	s, _ := v.(string)
	return s
}

// collectTableMiddleware is a no-op TableMiddleware whose presence makes the
// TableProcessor retain rows in its Table (the processor discards rows when no
// table middleware is registered).
type collectTableMiddleware struct{}

func (collectTableMiddleware) Process(_ context.Context, table *types.Table) (*types.Table, error) {
	return table, nil
}
func (collectTableMiddleware) Close(_ context.Context) error { return nil }
