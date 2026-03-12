package runtime

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jayl2kor/skillhub/internal/skill"
)

func TestPromptRunner(t *testing.T) {
	dir := t.TempDir()
	promptContent := "# Hello World\nThis is a test prompt.\n"

	if err := os.WriteFile(filepath.Join(dir, "prompt.md"), []byte(promptContent), 0644); err != nil {
		t.Fatal(err)
	}

	s := skill.InstalledSkill{
		Manifest: skill.Manifest{
			Name:  "test",
			Entry: "prompt.md",
			Type:  "prompt",
		},
		Dir: dir,
	}

	runner := &PromptRunner{}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runner.Run(context.Background(), s, nil)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	if buf.String() != promptContent {
		t.Errorf("expected %q, got %q", promptContent, buf.String())
	}
}

func TestPromptRunnerPathEscape(t *testing.T) {
	dir := t.TempDir()

	s := skill.InstalledSkill{
		Manifest: skill.Manifest{
			Name:  "test",
			Entry: "../../../etc/passwd",
			Type:  "prompt",
		},
		Dir: dir,
	}

	runner := &PromptRunner{}
	err := runner.Run(context.Background(), s, nil)
	if err == nil {
		t.Error("expected error for path escape")
	}
}

func TestPromptRunnerMissingFile(t *testing.T) {
	dir := t.TempDir()

	s := skill.InstalledSkill{
		Manifest: skill.Manifest{
			Name:  "test",
			Entry: "nonexistent.md",
			Type:  "prompt",
		},
		Dir: dir,
	}

	runner := &PromptRunner{}
	err := runner.Run(context.Background(), s, nil)
	if err == nil {
		t.Error("expected error for missing file")
	}
}
