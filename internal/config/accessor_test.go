package config

import (
	"testing"
)

func testConfig() *Config {
	return &Config{
		InstallDir: "/home/user/.skillhub/skills",
		CacheDir:   "/home/user/.skillhub/cache",
		LogDir:     "/home/user/.skillhub/logs",
		Registries: []RegistryEntry{
			{
				Name:         "myrepo",
				URL:          "https://github.com/org/repo",
				Token:        "ghp_abcdef1234567890",
				Username:     "user1",
				Branch:       "main",
				SkillsPrefix: ".claude/skills",
			},
			{
				Name: "other",
				URL:  "https://github.com/org/other",
			},
		},
	}
}

func TestGetValueTopLevel(t *testing.T) {
	cfg := testConfig()

	tests := []struct {
		key  string
		want string
	}{
		{"install_dir", "/home/user/.skillhub/skills"},
		{"cache_dir", "/home/user/.skillhub/cache"},
		{"log_dir", "/home/user/.skillhub/logs"},
	}

	for _, tt := range tests {
		val, err := cfg.GetValue(tt.key)
		if err != nil {
			t.Errorf("GetValue(%q): %v", tt.key, err)
		}
		if val != tt.want {
			t.Errorf("GetValue(%q) = %q, want %q", tt.key, val, tt.want)
		}
	}
}

func TestGetValueRegistryByName(t *testing.T) {
	cfg := testConfig()

	val, err := cfg.GetValue("registries.myrepo.url")
	if err != nil {
		t.Fatalf("GetValue: %v", err)
	}
	if val != "https://github.com/org/repo" {
		t.Errorf("got %q, want %q", val, "https://github.com/org/repo")
	}

	val, err = cfg.GetValue("registries.myrepo.skills_prefix")
	if err != nil {
		t.Fatalf("GetValue: %v", err)
	}
	if val != ".claude/skills" {
		t.Errorf("got %q, want %q", val, ".claude/skills")
	}
}

func TestGetValueRegistryByIndex(t *testing.T) {
	cfg := testConfig()

	val, err := cfg.GetValue("registries.0.url")
	if err != nil {
		t.Fatalf("GetValue: %v", err)
	}
	if val != "https://github.com/org/repo" {
		t.Errorf("got %q, want %q", val, "https://github.com/org/repo")
	}

	val, err = cfg.GetValue("registries.1.name")
	if err != nil {
		t.Fatalf("GetValue: %v", err)
	}
	if val != "other" {
		t.Errorf("got %q, want %q", val, "other")
	}
}

func TestGetValueErrors(t *testing.T) {
	cfg := testConfig()

	errorCases := []string{
		"nonexistent",
		"registries",
		"registries.myrepo",
		"registries.nosuch.url",
		"registries.99.url",
		"registries.myrepo.nonexistent",
		"install_dir.sub",
	}

	for _, key := range errorCases {
		_, err := cfg.GetValue(key)
		if err == nil {
			t.Errorf("GetValue(%q) expected error, got nil", key)
		}
	}
}

func TestSetValueTopLevel(t *testing.T) {
	cfg := testConfig()

	if err := cfg.SetValue("install_dir", "/new/path"); err != nil {
		t.Fatalf("SetValue: %v", err)
	}
	if cfg.InstallDir != "/new/path" {
		t.Errorf("got %q, want %q", cfg.InstallDir, "/new/path")
	}
}

func TestSetValueRegistryByName(t *testing.T) {
	cfg := testConfig()

	if err := cfg.SetValue("registries.myrepo.skills_prefix", "custom/prefix"); err != nil {
		t.Fatalf("SetValue: %v", err)
	}
	if cfg.Registries[0].SkillsPrefix != "custom/prefix" {
		t.Errorf("got %q, want %q", cfg.Registries[0].SkillsPrefix, "custom/prefix")
	}
}

func TestSetValueErrors(t *testing.T) {
	cfg := testConfig()

	errorCases := []struct {
		key   string
		value string
	}{
		{"nonexistent", "val"},
		{"registries", "val"},
		{"registries.myrepo", "val"},
		{"registries.nosuch.url", "val"},
		{"registries.myrepo.nonexistent", "val"},
	}

	for _, tt := range errorCases {
		err := cfg.SetValue(tt.key, tt.value)
		if err == nil {
			t.Errorf("SetValue(%q, %q) expected error, got nil", tt.key, tt.value)
		}
	}
}

func TestListValues(t *testing.T) {
	cfg := testConfig()
	kvs := cfg.ListValues()

	// 3 top-level + 2 registries * 6 fields = 15
	if len(kvs) != 15 {
		t.Fatalf("ListValues returned %d entries, want 15", len(kvs))
	}

	// First 3 should be top-level
	if kvs[0].Key != "install_dir" {
		t.Errorf("first key = %q, want install_dir", kvs[0].Key)
	}

	// Check a registry entry
	if kvs[3].Key != "registries.myrepo.name" {
		t.Errorf("kvs[3].Key = %q, want registries.myrepo.name", kvs[3].Key)
	}
}

func TestListValuesEmpty(t *testing.T) {
	cfg := &Config{InstallDir: "/a", CacheDir: "/b", LogDir: "/c"}
	kvs := cfg.ListValues()
	if len(kvs) != 3 {
		t.Fatalf("ListValues returned %d entries, want 3", len(kvs))
	}
}
