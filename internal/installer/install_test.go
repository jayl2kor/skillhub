package installer

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jayl2kor/skillhub/internal/config"
	"github.com/jayl2kor/skillhub/internal/registry"
	"github.com/jayl2kor/skillhub/internal/storage"
)

func setupTestRegistry(t *testing.T) (string, string) {
	t.Helper()

	regDir := t.TempDir()

	idx := registry.Index{
		Skills: []registry.IndexEntry{
			{
				Name:        "test-skill",
				Version:     "1.0.0",
				Description: "A test skill",
				Tags:        []string{"test"},
				DownloadURL: "packages/test-skill-1.0.0.tar.gz",
			},
		},
	}

	idxData, err := json.Marshal(idx)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(regDir, "index.json"), idxData, 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(regDir, "packages"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create tar.gz with skill files
	archivePath := filepath.Join(regDir, "packages", "test-skill-1.0.0.tar.gz")
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	files := map[string]string{
		"skill.json": `{"name":"test-skill","version":"1.0.0","description":"A test skill","entry":"prompt.md","type":"prompt"}`,
		"prompt.md":  "# Test\n",
		"SKILL.md":   "# Test Skill\nA test skill for Claude Code.\n",
	}

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}

	tw.Close()
	gw.Close()
	f.Close()

	homeDir := t.TempDir()
	return regDir, homeDir
}

func TestInstall(t *testing.T) {
	regDir, homeDir := setupTestRegistry(t)

	paths := storage.NewPaths(homeDir)
	if err := paths.EnsureDirectories(); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Registries: []config.RegistryEntry{
			{Name: "test-reg", URL: regDir},
		},
		InstallDir: paths.SkillsDir,
		CacheDir:   paths.CacheDir,
		LogDir:     paths.LogDir,
	}

	inst := NewInstaller(paths, cfg)

	if err := inst.Install(context.Background(), "test-skill", false, false); err != nil {
		t.Fatalf("Install: %v", err)
	}

	if !storage.IsInstalled(paths, "test-skill") {
		t.Error("skill should be installed")
	}

	s, err := storage.GetInstalledSkill(paths, "test-skill")
	if err != nil {
		t.Fatalf("GetInstalledSkill: %v", err)
	}
	if s.Manifest.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", s.Manifest.Version)
	}
}

func TestInstallAlreadyInstalled(t *testing.T) {
	regDir, homeDir := setupTestRegistry(t)

	paths := storage.NewPaths(homeDir)
	if err := paths.EnsureDirectories(); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Registries: []config.RegistryEntry{
			{Name: "test-reg", URL: regDir},
		},
	}

	inst := NewInstaller(paths, cfg)

	if err := inst.Install(context.Background(), "test-skill", false, false); err != nil {
		t.Fatalf("first Install: %v", err)
	}

	err := inst.Install(context.Background(), "test-skill", false, false)
	if err == nil {
		t.Error("expected error for already installed skill")
	}

	if err := inst.Install(context.Background(), "test-skill", true, false); err != nil {
		t.Errorf("force Install: %v", err)
	}
}

func TestInstallNotFound(t *testing.T) {
	regDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(regDir, "index.json"), []byte(`{"skills": []}`), 0644); err != nil {
		t.Fatal(err)
	}

	homeDir := t.TempDir()
	paths := storage.NewPaths(homeDir)
	if err := paths.EnsureDirectories(); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Registries: []config.RegistryEntry{
			{Name: "empty", URL: regDir},
		},
	}

	inst := NewInstaller(paths, cfg)
	err := inst.Install(context.Background(), "nonexistent", false, false)
	if err == nil {
		t.Error("expected error for nonexistent skill")
	}
}

func setupTestDirectoryRegistry(t *testing.T) (string, string) {
	t.Helper()

	regDir := t.TempDir()

	// Create skill directory (no tar.gz)
	skillDir := filepath.Join(regDir, "skills", "dir-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"skill.json": `{"name":"dir-skill","version":"2.0.0","description":"A directory skill","entry":"prompt.md","type":"prompt"}`,
		"prompt.md":  "# Directory Test\n",
		"SKILL.md":   "# Dir Skill\nA directory-based skill.\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(skillDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create index.json with directory-based download_url
	idx := registry.Index{
		Skills: []registry.IndexEntry{
			{
				Name:        "dir-skill",
				Version:     "2.0.0",
				Description: "A directory skill",
				Tags:        []string{"test"},
				DownloadURL: "skills/dir-skill/",
			},
		},
	}
	idxData, err := json.Marshal(idx)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(regDir, "index.json"), idxData, 0644); err != nil {
		t.Fatal(err)
	}

	homeDir := t.TempDir()
	return regDir, homeDir
}

func TestInstallDirectoryMode(t *testing.T) {
	regDir, homeDir := setupTestDirectoryRegistry(t)

	paths := storage.NewPaths(homeDir)
	if err := paths.EnsureDirectories(); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Registries: []config.RegistryEntry{
			{Name: "dir-reg", URL: regDir},
		},
	}

	inst := NewInstaller(paths, cfg)

	if err := inst.Install(context.Background(), "dir-skill", false, false); err != nil {
		t.Fatalf("Install directory mode: %v", err)
	}

	if !storage.IsInstalled(paths, "dir-skill") {
		t.Error("skill should be installed")
	}

	s, err := storage.GetInstalledSkill(paths, "dir-skill")
	if err != nil {
		t.Fatalf("GetInstalledSkill: %v", err)
	}
	if s.Manifest.Version != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", s.Manifest.Version)
	}
}

func TestInstallDirectoryModeForceReinstall(t *testing.T) {
	regDir, homeDir := setupTestDirectoryRegistry(t)

	paths := storage.NewPaths(homeDir)
	if err := paths.EnsureDirectories(); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Registries: []config.RegistryEntry{
			{Name: "dir-reg", URL: regDir},
		},
	}

	inst := NewInstaller(paths, cfg)

	if err := inst.Install(context.Background(), "dir-skill", false, false); err != nil {
		t.Fatalf("first Install: %v", err)
	}

	// Should fail without --force
	err := inst.Install(context.Background(), "dir-skill", false, false)
	if err == nil {
		t.Error("expected error for already installed skill")
	}

	// Should succeed with --force
	if err := inst.Install(context.Background(), "dir-skill", true, false); err != nil {
		t.Errorf("force Install directory mode: %v", err)
	}
}

func TestInstallNoRegistries(t *testing.T) {
	homeDir := t.TempDir()
	paths := storage.NewPaths(homeDir)
	if err := paths.EnsureDirectories(); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{}

	inst := NewInstaller(paths, cfg)
	err := inst.Install(context.Background(), "test", false, false)
	if err == nil {
		t.Error("expected error when no registries configured")
	}
}
