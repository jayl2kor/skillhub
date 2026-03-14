package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jayl2kor/skillhub/internal/skill"
)

// PromptRunner runs prompt-type skills by printing the entry file contents.
type PromptRunner struct{}

// Run outputs the content of the skill's entry file to stdout.
func (r *PromptRunner) Run(_ context.Context, s skill.InstalledSkill, _ []string) error {
	entry := s.Manifest.Entry

	// Validate entry path
	cleaned := filepath.Clean(entry)
	if filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, "..") {
		return fmt.Errorf("invalid entry path: %s", entry)
	}

	entryPath := filepath.Join(s.Dir, cleaned)

	// Verify the resolved path is within skill directory
	rel, err := filepath.Rel(s.Dir, entryPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("entry path escapes skill directory: %s", entry)
	}

	data, err := os.ReadFile(entryPath)
	if err != nil {
		return fmt.Errorf("reading entry file: %w", err)
	}

	fmt.Print(string(data))
	return nil
}
