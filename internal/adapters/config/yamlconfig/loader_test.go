package yamlconfig

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoader_LoadDefinitions_OK(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "services.yaml")
	content := `
services:
  - name: Test
    endpoint: https://example.com
    interval: 45s
    timeout: 10s
    retries: 1
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(path)
	defs, err := l.LoadDefinitions(context.Background())
	if err != nil {
		t.Fatalf("LoadDefinitions: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("len = %d", len(defs))
	}
	if defs[0].Name != "Test" || defs[0].Endpoint != "https://example.com" {
		t.Fatalf("def = %+v", defs[0])
	}
	if defs[0].IntervalSeconds != 45 || defs[0].TimeoutSeconds != 10 || defs[0].ExtraRetryAttempts != 1 {
		t.Fatalf("interval/timeout/retries = %d/%d/%d", defs[0].IntervalSeconds, defs[0].TimeoutSeconds, defs[0].ExtraRetryAttempts)
	}
}

func TestLoader_LoadDefinitions_invalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(`services: [`), 0o600); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(path)
	_, err := l.LoadDefinitions(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoader_LoadDefinitions_validationError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.yaml")
	content := `
services:
  - name: ""
    endpoint: https://example.com
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(path)
	_, err := l.LoadDefinitions(context.Background())
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLoader_LoadDefinitions_missingFile(t *testing.T) {
	l := NewLoader(filepath.Join(t.TempDir(), "nope.yaml"))
	_, err := l.LoadDefinitions(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want ErrNotExist, got %v", err)
	}
}
