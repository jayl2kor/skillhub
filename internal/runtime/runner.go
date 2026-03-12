package runtime

import (
	"context"
	"fmt"

	"github.com/jayl2kor/skillhub/internal/skill"
)

type SkillRunner interface {
	Run(ctx context.Context, s skill.InstalledSkill, args []string) error
}

func RunnerFor(skillType string) (SkillRunner, error) {
	switch skillType {
	case "prompt":
		return &PromptRunner{}, nil
	default:
		return nil, fmt.Errorf("unsupported skill type: %s", skillType)
	}
}
