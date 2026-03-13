package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("/home/test")

	if len(cfg.Registries) != 0 {
		t.Errorf("expected empty registries, got %d", len(cfg.Registries))
	}
	if cfg.InstallDir != "/home/test/skills" {
		t.Errorf("unexpected install dir: %s", cfg.InstallDir)
	}
}

func TestLoadSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		Registries: []RegistryEntry{
			{Name: "test", URL: "https://github.com/org/repo"},
		},
		InstallDir: "/tmp/skills",
		CacheDir:   "/tmp/cache",
		LogDir:     "/tmp/logs",
	}

	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded.Registries) != 1 {
		t.Fatalf("expected 1 registry, got %d", len(loaded.Registries))
	}
	if loaded.Registries[0].Name != "test" {
		t.Errorf("expected registry name 'test', got %q", loaded.Registries[0].Name)
	}
	if loaded.InstallDir != "/tmp/skills" {
		t.Errorf("expected install dir '/tmp/skills', got %q", loaded.InstallDir)
	}
}

func TestLoadInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte("not: [valid: yaml:"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadMissing(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestValidate(t *testing.T) {
	cfg := &Config{
		Registries: []RegistryEntry{
			{Name: "", URL: "https://example.com"},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty registry name")
	}

	cfg.Registries[0].Name = "test"
	cfg.Registries[0].URL = ""
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty registry URL")
	}

	cfg.Registries[0].URL = "https://example.com"
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAddRemoveRegistry(t *testing.T) {
	cfg := &Config{}

	if err := cfg.AddRegistry("test", "https://example.com", "", "", "", ""); err != nil {
		t.Fatalf("AddRegistry: %v", err)
	}

	if len(cfg.Registries) != 1 {
		t.Fatalf("expected 1 registry, got %d", len(cfg.Registries))
	}

	// Duplicate name should update URL, token, and username
	if err := cfg.AddRegistry("test", "https://other.com", "tok", "user1", "", ""); err != nil {
		t.Fatalf("AddRegistry update: %v", err)
	}
	if len(cfg.Registries) != 1 {
		t.Fatalf("expected 1 registry after update, got %d", len(cfg.Registries))
	}
	if cfg.Registries[0].URL != "https://other.com" || cfg.Registries[0].Token != "tok" || cfg.Registries[0].Username != "user1" {
		t.Fatalf("expected updated URL, token, and username, got %+v", cfg.Registries[0])
	}

	if err := cfg.RemoveRegistry("test"); err != nil {
		t.Fatalf("RemoveRegistry: %v", err)
	}

	if len(cfg.Registries) != 0 {
		t.Errorf("expected 0 registries, got %d", len(cfg.Registries))
	}

	// Remove non-existent
	if err := cfg.RemoveRegistry("test"); err == nil {
		t.Error("expected error for non-existent registry")
	}
}
