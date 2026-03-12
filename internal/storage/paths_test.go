package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewPaths(t *testing.T) {
	p := NewPaths("/test/home")

	if p.Home != "/test/home" {
		t.Errorf("expected Home '/test/home', got %q", p.Home)
	}
	if p.Config != "/test/home/config.yaml" {
		t.Errorf("expected Config '/test/home/config.yaml', got %q", p.Config)
	}
	if p.SkillsDir != "/test/home/skills" {
		t.Errorf("expected SkillsDir '/test/home/skills', got %q", p.SkillsDir)
	}
}

func TestEnsureDirectories(t *testing.T) {
	dir := t.TempDir()
	p := NewPaths(filepath.Join(dir, "skillhub"))

	if err := p.EnsureDirectories(); err != nil {
		t.Fatalf("EnsureDirectories: %v", err)
	}

	for _, d := range []string{p.Home, p.SkillsDir, p.CacheDir, p.LogDir, p.TmpDir} {
		info, err := os.Stat(d)
		if err != nil {
			t.Errorf("directory %s not created: %v", d, err)
		} else if !info.IsDir() {
			t.Errorf("%s is not a directory", d)
		}
	}

	// Should be idempotent
	if err := p.EnsureDirectories(); err != nil {
		t.Errorf("EnsureDirectories (idempotent): %v", err)
	}
}

func TestSkillDir(t *testing.T) {
	p := NewPaths("/test/home")
	got := p.SkillDir("code-review")
	expected := "/test/home/skills/code-review"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestDefaultHome(t *testing.T) {
	// Test with environment variable
	os.Setenv("SKILLHUB_HOME", "/custom/path")
	defer os.Unsetenv("SKILLHUB_HOME")

	home := DefaultHome()
	if home != "/custom/path" {
		t.Errorf("expected '/custom/path', got %q", home)
	}
}
