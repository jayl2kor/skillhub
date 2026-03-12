package cli

import (
	"context"
	"fmt"

	"github.com/jayl2kor/skillhub/internal/runtime"
	"github.com/jayl2kor/skillhub/internal/skill"
	"github.com/jayl2kor/skillhub/internal/storage"

	"github.com/spf13/cobra"
)

var toolFlag string

var runCmd = &cobra.Command{
	Use:   "run <skill>",
	Short: "Run an installed skill",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		skillArgs := args[1:]

		// Validate --tool flag if provided
		if toolFlag != "" {
			if _, err := skill.LookupAgent(toolFlag); err != nil {
				return err
			}
		}

		logVerbose("loading skill %q", name)
		s, err := storage.GetInstalledSkill(paths, name)
		if err != nil {
			return fmt.Errorf("skill %q is not installed", name)
		}

		// Check agent compatibility if manifest specifies compatible_agents
		if toolFlag != "" && len(s.Manifest.CompatibleAgents) > 0 {
			compatible := false
			for _, a := range s.Manifest.CompatibleAgents {
				if a == toolFlag {
					compatible = true
					break
				}
			}
			if !compatible {
				return fmt.Errorf("skill %q is not compatible with agent %q (compatible: %v)", name, toolFlag, s.Manifest.CompatibleAgents)
			}
		}

		logVerbose("skill type: %s, entry: %s", s.Manifest.Type, s.Manifest.Entry)
		runner, err := runtime.RunnerFor(s.Manifest.Type)
		if err != nil {
			return err
		}

		logVerbose("executing with runner for type %q", s.Manifest.Type)
		return runner.Run(context.Background(), *s, skillArgs)
	},
}

func init() {
	runCmd.Flags().StringVar(&toolFlag, "tool", "", "agent type (claude, cursor, windsurf, cline, generic)")
	rootCmd.AddCommand(runCmd)
}
