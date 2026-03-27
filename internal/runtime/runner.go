// Package runtime provides runners that execute installed skills.
package runtime

import (
	"context"
	"fmt"

	"github.com/jayl2kor/skillhub/internal/skill"
)

// SkillRunner executes an installed skill with the given arguments.
type SkillRunner interface {
	Run(ctx context.Context, s skill.InstalledSkill, args []string) error
}

// RunnerFor returns the appropriate SkillRunner for the given skill type.
func RunnerFor(skillType string) (SkillRunner, error) {
	switch skillType {
	case "prompt":
		return &PromptRunner{}, nil
	case "shell":
		return &ExecRunner{Command: "bash"}, nil
	case "python":
		return &ExecRunner{Command: "python3"}, nil
	case "node":
		return &ExecRunner{Command: "node"}, nil
	default:
		return nil, fmt.Errorf("unsupported skill type: %s", skillType)
	}
}
