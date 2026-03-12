package cli

import (
	"context"
	"fmt"

	"github.com/jayl2kor/skillhub/internal/runtime"
	"github.com/jayl2kor/skillhub/internal/storage"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <skill>",
	Short: "Run an installed skill",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		skillArgs := args[1:]

		logVerbose("loading skill %q", name)
		s, err := storage.GetInstalledSkill(paths, name)
		if err != nil {
			return fmt.Errorf("skill %q is not installed", name)
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
	rootCmd.AddCommand(runCmd)
}
