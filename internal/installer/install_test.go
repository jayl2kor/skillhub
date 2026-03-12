package installer

import (
	"archive/tar"
	"compress/gzip"
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

	idxData, _ := json.Marshal(idx)
	os.WriteFile(filepath.Join(regDir, "index.json"), idxData, 0644)

	os.MkdirAll(filepath.Join(regDir, "packages"), 0755)

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
		tw.WriteHeader(hdr)
		tw.Write([]byte(content))
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
	paths.EnsureDirectories()

	cfg := &config.Config{
		Registries: []config.RegistryEntry{
			{Name: "test-reg", URL: regDir},
		},
		InstallDir: paths.SkillsDir,
		CacheDir:   paths.CacheDir,
		LogDir:     paths.LogDir,
	}

	inst := NewInstaller(paths, cfg)

	if err := inst.Install("test-skill", false, false); err != nil {
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
	paths.EnsureDirectories()

	cfg := &config.Config{
		Registries: []config.RegistryEntry{
			{Name: "test-reg", URL: regDir},
		},
	}

	inst := NewInstaller(paths, cfg)

	if err := inst.Install("test-skill", false, false); err != nil {
		t.Fatalf("first Install: %v", err)
	}

	err := inst.Install("test-skill", false, false)
	if err == nil {
		t.Error("expected error for already installed skill")
	}

	if err := inst.Install("test-skill", true, false); err != nil {
		t.Errorf("force Install: %v", err)
	}
}

func TestInstallNotFound(t *testing.T) {
	regDir := t.TempDir()
	os.WriteFile(filepath.Join(regDir, "index.json"), []byte(`{"skills": []}`), 0644)

	homeDir := t.TempDir()
	paths := storage.NewPaths(homeDir)
	paths.EnsureDirectories()

	cfg := &config.Config{
		Registries: []config.RegistryEntry{
			{Name: "empty", URL: regDir},
		},
	}

	inst := NewInstaller(paths, cfg)
	err := inst.Install("nonexistent", false, false)
	if err == nil {
		t.Error("expected error for nonexistent skill")
	}
}

func TestInstallNoRegistries(t *testing.T) {
	homeDir := t.TempDir()
	paths := storage.NewPaths(homeDir)
	paths.EnsureDirectories()

	cfg := &config.Config{}

	inst := NewInstaller(paths, cfg)
	err := inst.Install("test", false, false)
	if err == nil {
		t.Error("expected error when no registries configured")
	}
}
