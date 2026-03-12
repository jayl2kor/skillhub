package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jayl2kor/skillhub/internal/config"
	"github.com/jayl2kor/skillhub/internal/skill"
	"github.com/jayl2kor/skillhub/internal/storage"
)

func setupTestHome(t *testing.T) *storage.Paths {
	t.Helper()
	dir := t.TempDir()
	p := storage.NewPaths(dir)
	p.ProjectRoot = dir // isolate from real project root
	if err := p.EnsureDirectories(); err != nil {
		t.Fatalf("creating test dirs: %v", err)
	}
	// Set global paths so commands use the test directory
	paths = p
	homeDir = dir
	return p
}

func writeTestConfig(t *testing.T, p *storage.Paths) *config.Config {
	t.Helper()
	cfg := config.DefaultConfig(p.Home)
	if err := cfg.Save(p.Config); err != nil {
		t.Fatalf("writing test config: %v", err)
	}
	return cfg
}

func installFakeSkill(t *testing.T, p *storage.Paths, name string) {
	t.Helper()
	dir := p.SkillDir(name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("creating skill dir: %v", err)
	}
	manifest := skill.Manifest{
		Name:        name,
		Version:     "1.0.0",
		Description: "test skill",
		Entry:       "prompt.md",
		Type:        "prompt",
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshaling manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skill.json"), data, 0644); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "prompt.md"), []byte("test prompt"), 0644); err != nil {
		t.Fatalf("writing prompt: %v", err)
	}
}

func TestInitCreatesWorkspace(t *testing.T) {
	dir := t.TempDir()
	paths = storage.NewPaths(dir)
	homeDir = dir

	err := initCmd.RunE(initCmd, nil)
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	// Verify directories exist
	for _, d := range []string{paths.SkillsDir, paths.CacheDir, paths.LogDir} {
		info, err := os.Stat(d)
		if err != nil {
			t.Errorf("directory %s not created: %v", d, err)
		} else if !info.IsDir() {
			t.Errorf("%s is not a directory", d)
		}
	}

	// Verify config was created
	if _, err := os.Stat(paths.Config); err != nil {
		t.Errorf("config not created: %v", err)
	}
}

func TestInitIdempotent(t *testing.T) {
	dir := t.TempDir()
	paths = storage.NewPaths(dir)
	homeDir = dir

	// Run init twice
	if err := initCmd.RunE(initCmd, nil); err != nil {
		t.Fatalf("first init: %v", err)
	}
	if err := initCmd.RunE(initCmd, nil); err != nil {
		t.Fatalf("second init: %v", err)
	}
}

func TestDoctorHealthyWorkspace(t *testing.T) {
	p := setupTestHome(t)
	writeTestConfig(t, p)

	err := doctorCmd.RunE(doctorCmd, nil)
	if err != nil {
		t.Fatalf("doctor should pass on healthy workspace: %v", err)
	}
}

func TestDoctorMissingConfig(t *testing.T) {
	dir := t.TempDir()
	paths = storage.NewPaths(dir)
	homeDir = dir
	// Don't create config or directories

	err := doctorCmd.RunE(doctorCmd, nil)
	if err == nil {
		t.Fatal("doctor should fail with missing config")
	}
}

func TestListEmptySkills(t *testing.T) {
	p := setupTestHome(t)
	writeTestConfig(t, p)

	// Should not error, just print "No skills installed."
	err := listCmd.RunE(listCmd, nil)
	if err != nil {
		t.Fatalf("list with no skills: %v", err)
	}
}

func TestListWithSkills(t *testing.T) {
	p := setupTestHome(t)
	writeTestConfig(t, p)
	installFakeSkill(t, p, "test-skill")

	err := listCmd.RunE(listCmd, nil)
	if err != nil {
		t.Fatalf("list with skills: %v", err)
	}
}

func TestRemoveInstalledSkill(t *testing.T) {
	p := setupTestHome(t)
	writeTestConfig(t, p)
	installFakeSkill(t, p, "removable")

	err := removeCmd.RunE(removeCmd, []string{"removable"})
	if err != nil {
		t.Fatalf("remove: %v", err)
	}

	if storage.IsInstalled(p, "removable") {
		t.Error("skill should not be installed after removal")
	}
}

func TestRemoveNonexistentSkill(t *testing.T) {
	p := setupTestHome(t)
	writeTestConfig(t, p)

	err := removeCmd.RunE(removeCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("remove should fail for nonexistent skill")
	}
}

func TestRemoveInvalidName(t *testing.T) {
	p := setupTestHome(t)
	writeTestConfig(t, p)

	err := removeCmd.RunE(removeCmd, []string{"../escape"})
	if err == nil {
		t.Fatal("remove should reject path traversal")
	}
}

func TestInfoNonexistentSkill(t *testing.T) {
	p := setupTestHome(t)
	writeTestConfig(t, p)

	err := infoCmd.RunE(infoCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("info should fail for nonexistent skill")
	}
}

func TestRunNonexistentSkill(t *testing.T) {
	p := setupTestHome(t)
	writeTestConfig(t, p)

	err := runCmd.RunE(runCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("run should fail for nonexistent skill")
	}
}

func TestRepoListEmpty(t *testing.T) {
	p := setupTestHome(t)
	writeTestConfig(t, p)

	err := repoListCmd.RunE(repoListCmd, nil)
	if err != nil {
		t.Fatalf("repo list with no registries: %v", err)
	}
}

func TestRepoRemoveNonexistent(t *testing.T) {
	p := setupTestHome(t)
	writeTestConfig(t, p)

	err := repoRemoveCmd.RunE(repoRemoveCmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("repo remove should fail for nonexistent registry")
	}
}
