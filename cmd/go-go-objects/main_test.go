package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewRootCommandIncludesGlazedHelp(t *testing.T) {
	root, err := newRootCommand()
	if err != nil {
		t.Fatalf("newRootCommand() error = %v", err)
	}
	if root.Use != "go-go-objects" {
		t.Fatalf("root.Use = %q, want go-go-objects", root.Use)
	}
	if root.PersistentFlags().Lookup("log-level") == nil {
		t.Fatal("expected log-level flag from Glazed logging section")
	}
	if root.Commands() == nil {
		t.Fatal("expected child commands")
	}
	foundHelp := false
	for _, cmd := range root.Commands() {
		if cmd.Name() == "help" {
			foundHelp = true
			break
		}
	}
	if !foundHelp {
		t.Fatal("expected Glazed help command")
	}
}

func TestParseDurationFlag(t *testing.T) {
	d, err := parseDurationFlag("cpu-timeout", "250ms")
	if err != nil {
		t.Fatalf("parseDurationFlag() error = %v", err)
	}
	if d != 250*time.Millisecond {
		t.Fatalf("duration = %v, want 250ms", d)
	}
	if _, err := parseDurationFlag("cpu-timeout", "not-a-duration"); err == nil {
		t.Fatal("expected invalid duration error")
	}
}

func TestLoadInputsUsesBuiltInDemoWhenPathsOmitted(t *testing.T) {
	manifest, bundle, err := loadInputs("", "")
	if err != nil {
		t.Fatalf("loadInputs() error = %v", err)
	}
	if !manifest.IsZero() {
		t.Fatalf("manifest = %#v, want zero manifest for derived built-in demo", manifest)
	}
	if bundle == "" {
		t.Fatal("expected built-in bundle")
	}
}

func TestLoadInputsAllowsBundleWithoutManifest(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "objects.js")
	if err := os.WriteFile(bundlePath, []byte(`exports.objects = {};`), 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	manifest, bundle, err := loadInputs(bundlePath, "")
	if err != nil {
		t.Fatalf("loadInputs() error = %v", err)
	}
	if !manifest.IsZero() {
		t.Fatalf("manifest = %#v, want zero manifest for derived namespaces", manifest)
	}
	if bundle != `exports.objects = {};` {
		t.Fatalf("bundle = %q", bundle)
	}
}

func TestLoadInputsRejectsManifestWithoutBundle(t *testing.T) {
	if _, _, err := loadInputs("", "durableobjects.yaml"); err == nil {
		t.Fatal("expected error when only manifest is provided")
	}
}

func TestLoadInputsReadsBundleAndYAMLManifest(t *testing.T) {
	dir := t.TempDir()
	bundlePath := filepath.Join(dir, "objects.js")
	manifestPath := filepath.Join(dir, "durableobjects.yaml")
	if err := os.WriteFile(bundlePath, []byte(`exports.objects = {};`), 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	if err := os.WriteFile(manifestPath, []byte("objects:\n  COUNTER: Counter\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	manifest, bundle, err := loadInputs(bundlePath, manifestPath)
	if err != nil {
		t.Fatalf("loadInputs() error = %v", err)
	}
	if bundle != `exports.objects = {};` {
		t.Fatalf("bundle = %q", bundle)
	}
	if class, ok := manifest.ClassForNamespace("COUNTER"); !ok || class != "Counter" {
		t.Fatalf("manifest COUNTER = %q ok=%v", class, ok)
	}
}
