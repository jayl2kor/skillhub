package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/jayl2kor/skillhub/internal/storage"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <skill>",
	Short: "Remove an installed skill",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Validate name (no path separators)
		if strings.ContainsAny(name, "/\\") {
			return fmt.Errorf("invalid skill name: %q", name)
		}

		if !storage.IsInstalled(paths, name) {
			return fmt.Errorf("skill %q is not installed", name)
		}

		skillDir := paths.SkillDir(name)
		if err := os.RemoveAll(skillDir); err != nil {
			return fmt.Errorf("removing skill: %w", err)
		}

		fmt.Printf("Removed %s\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
