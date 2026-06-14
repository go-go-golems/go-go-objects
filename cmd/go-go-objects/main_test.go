package main

import (
	"os"
	"path/filepath"
	"testing"
)

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
