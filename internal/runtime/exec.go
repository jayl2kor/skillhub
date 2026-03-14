package runtime

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jayl2kor/skillhub/internal/skill"
)

// ExecRunner runs a skill's entry file using an external interpreter.
type ExecRunner struct {
	Command string   // e.g. "bash", "python3", "node"
	Args    []string // extra args before the script path
}

// Run executes the skill's entry file using the configured interpreter.
func (r *ExecRunner) Run(ctx context.Context, s skill.InstalledSkill, args []string) error {
	entry := s.Manifest.Entry

	cleaned := filepath.Clean(entry)
	if filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, "..") {
		return fmt.Errorf("invalid entry path: %s", entry)
	}

	entryPath := filepath.Join(s.Dir, cleaned)

	rel, err := filepath.Rel(s.Dir, entryPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("entry path escapes skill directory: %s", entry)
	}

	if _, err := os.Stat(entryPath); err != nil {
		return fmt.Errorf("entry file not found: %w", err)
	}

	cmdArgs := append(r.Args, entryPath)
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, r.Command, cmdArgs...)
	cmd.Dir = s.Dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
