package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jayl2kor/skillhub/internal/skill"

	"github.com/spf13/cobra"
)

var lintStrict bool

var lintCmd = &cobra.Command{
	Use:   "lint [dir]",
	Short: "Validate skill structure",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}

		var errors []string
		var warnings []string

		// Check skill.json exists
		manifestPath := filepath.Join(dir, "skill.json")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			errors = append(errors, "skill.json not found")
			printIssues(errors, warnings)
			return fmt.Errorf("lint failed with %d error(s)", len(errors))
		}

		// Load manifest
		m, err := skill.LoadManifest(manifestPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("skill.json: %v", err))
			printIssues(errors, warnings)
			return fmt.Errorf("lint failed with %d error(s)", len(errors))
		}

		// Validate manifest
		if err := m.Validate(); err != nil {
			errors = append(errors, fmt.Sprintf("skill.json: %v", err))
		}

		// Check entry file exists
		if m.Entry != "" {
			entryPath := filepath.Join(dir, m.Entry)
			if _, err := os.Stat(entryPath); os.IsNotExist(err) {
				errors = append(errors, fmt.Sprintf("entry file %q not found", m.Entry))
			}
		}

		// Warnings
		if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); os.IsNotExist(err) {
			warnings = append(warnings, "SKILL.md not found (needed for --global install)")
		}
		if m.Author == "" {
			warnings = append(warnings, "author field is empty")
		}

		printIssues(errors, warnings)

		totalErrors := len(errors)
		if lintStrict {
			totalErrors += len(warnings)
		}

		if totalErrors > 0 {
			return fmt.Errorf("lint failed with %d issue(s)", totalErrors)
		}

		fmt.Println("No issues found.")
		return nil
	},
}

func init() {
	lintCmd.Flags().BoolVar(&lintStrict, "strict", false, "treat warnings as errors")
	rootCmd.AddCommand(lintCmd)
}

func printIssues(errors, warnings []string) {
	for _, e := range errors {
		fmt.Fprintf(os.Stderr, "[ERROR] %s\n", e)
	}
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "[WARN]  %s\n", w)
	}
}
