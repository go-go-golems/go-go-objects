package durableobjectsprovider

import (
	"sort"
	"testing"

	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

// TestRegisterExposesHelpSource asserts that Register contributes a
// HelpSource named "go-go-objects" whose embedded filesystem loads exactly the
// three Glazed help pages shipped by the provider. This is the contract that
// makes the docs bundleable into a generated xgoja binary via a buildspec
// `kind: help` source.
func TestRegisterExposesHelpSource(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("Register: %v", err)
	}

	source, ok := registry.ResolveHelpSource(PackageID, "go-go-objects")
	if !ok {
		t.Fatalf("ResolveHelpSource(%q, %q) not found", PackageID, "go-go-objects")
	}
	if source.FS == nil {
		t.Fatal("HelpSource FS is nil")
	}
	if source.Root == "" {
		t.Fatal("HelpSource Root is empty")
	}

	helpSystem := help.NewHelpSystem()
	if err := helpSystem.LoadSectionsFromFS(source.FS, source.Root); err != nil {
		t.Fatalf("LoadSectionsFromFS: %v", err)
	}

	sections, err := helpSystem.QuerySections("")
	if err != nil {
		t.Fatalf("QuerySections: %v", err)
	}

	got := make([]string, 0, len(sections))
	for _, s := range sections {
		if s == nil {
			continue
		}
		got = append(got, s.Slug)
	}
	sort.Strings(got)

	want := []string{
		"go-go-objects-js-api",
		"go-go-objects-overview",
		"go-go-objects-xgoja-provider",
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d loaded sections, got %d (%v)", len(want), len(got), got)
	}
	for i, slug := range want {
		if got[i] != slug {
			t.Fatalf("section %d: want slug %q, got %q (all=%v)", i, slug, got[i], got)
		}
	}
}

// TestRegisterExposesHelpSourceContent sanity-checks that a loaded section
// carries a non-empty body, so the embedded pages are not just frontmatter.
func TestRegisterExposesHelpSourceContent(t *testing.T) {
	registry := providerapi.NewProviderRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("Register: %v", err)
	}
	source, ok := registry.ResolveHelpSource(PackageID, "go-go-objects")
	if !ok {
		t.Fatalf("HelpSource not found")
	}

	helpSystem := help.NewHelpSystem()
	if err := helpSystem.LoadSectionsFromFS(source.FS, source.Root); err != nil {
		t.Fatalf("LoadSectionsFromFS: %v", err)
	}

	section, err := helpSystem.GetSectionWithSlug("go-go-objects-overview")
	if err != nil {
		t.Fatalf("GetSectionWithSlug: %v", err)
	}
	if section == nil {
		t.Fatal("section is nil")
	}
	if section.Title == "" {
		t.Error("section Title is empty")
	}
	if section.Content == "" {
		t.Error("section Content is empty")
	}
}
