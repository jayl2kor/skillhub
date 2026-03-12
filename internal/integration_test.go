package internal

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jayl2kor/skillhub/internal/config"
	"github.com/jayl2kor/skillhub/internal/installer"
	"github.com/jayl2kor/skillhub/internal/registry"
	"github.com/jayl2kor/skillhub/internal/runtime"
	"github.com/jayl2kor/skillhub/internal/storage"
)

func TestFullWorkflow(t *testing.T) {
	// Setup: create a local registry with one skill
	regDir := t.TempDir()
	homeDir := t.TempDir()

	// Create index.json
	idx := registry.Index{
		Skills: []registry.IndexEntry{
			{
				Name:        "test-prompt",
				Version:     "1.0.0",
				Description: "A test prompt skill",
				Tags:        []string{"test"},
				DownloadURL: "packages/test-prompt-1.0.0.tar.gz",
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

	// Create skill archive
	if err := os.MkdirAll(filepath.Join(regDir, "packages"), 0755); err != nil {
		t.Fatal(err)
	}
	createSkillArchive(t,
		filepath.Join(regDir, "packages", "test-prompt-1.0.0.tar.gz"),
		map[string]string{
			"skill.json": `{"name":"test-prompt","version":"1.0.0","description":"A test prompt skill","entry":"prompt.md","type":"prompt","tags":["test"]}`,
			"prompt.md":  "# Test Prompt\nHello from test prompt.\n",
			"SKILL.md":   "# Test Prompt\nA test prompt skill for Claude Code.\n",
		},
	)

	// Step 1: Init
	paths := storage.NewPaths(homeDir)
	paths.ProjectRoot = homeDir // isolate from real project root
	if err := paths.EnsureDirectories(); err != nil {
		t.Fatalf("EnsureDirectories: %v", err)
	}

	// Step 2: Create config with registry
	cfg := config.DefaultConfig(homeDir)
	if err := cfg.AddRegistry("test-reg", regDir, "", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Save(paths.Config); err != nil {
		t.Fatalf("Save config: %v", err)
	}

	// Step 3: Search
	sources := []registry.RepoSource{{Name: "test-reg", URL: regDir}}
	client := registry.NewClient()
	fetchedIdx, err := client.FetchAllIndexes(sources)
	if err != nil {
		t.Fatalf("FetchAllIndexes: %v", err)
	}

	results := fetchedIdx.Search("test")
	if len(results) != 1 {
		t.Fatalf("expected 1 search result, got %d", len(results))
	}
	if results[0].Name != "test-prompt" {
		t.Errorf("expected 'test-prompt', got %q", results[0].Name)
	}

	// Step 4: Install
	inst := installer.NewInstaller(paths, cfg)
	if err := inst.Install("test-prompt", false, false); err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Step 5: List
	skills, err := storage.ListInstalledSkills(paths)
	if err != nil {
		t.Fatalf("ListInstalledSkills: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 installed skill, got %d", len(skills))
	}
	if skills[0].Manifest.Name != "test-prompt" {
		t.Errorf("expected 'test-prompt', got %q", skills[0].Manifest.Name)
	}

	// Step 6: Info
	s, err := storage.GetInstalledSkill(paths, "test-prompt")
	if err != nil {
		t.Fatalf("GetInstalledSkill: %v", err)
	}
	if s.Manifest.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", s.Manifest.Version)
	}

	// Step 7: Run
	runner, err := runtime.RunnerFor("prompt")
	if err != nil {
		t.Fatalf("RunnerFor: %v", err)
	}

	// Redirect stdout to capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	if err := runner.Run(context.Background(), *s, nil); err != nil {
		w.Close()
		os.Stdout = oldStdout
		t.Fatalf("Run: %v", err)
	}

	w.Close()
	os.Stdout = oldStdout

	var buf [1024]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])
	if output != "# Test Prompt\nHello from test prompt.\n" {
		t.Errorf("unexpected output: %q", output)
	}

	// Step 8: Remove
	if err := os.RemoveAll(paths.SkillDir("test-prompt")); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}

	if storage.IsInstalled(paths, "test-prompt") {
		t.Error("skill should be removed")
	}

	// Step 9: List (empty)
	skills, err = storage.ListInstalledSkills(paths)
	if err != nil {
		t.Fatalf("ListInstalledSkills: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(skills))
	}

	// Step 10: Remove registry
	if err := cfg.RemoveRegistry("test-reg"); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Registries) != 0 {
		t.Errorf("expected 0 registries, got %d", len(cfg.Registries))
	}
}

func createSkillArchive(t *testing.T, archivePath string, files map[string]string) {
	t.Helper()

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

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
}
