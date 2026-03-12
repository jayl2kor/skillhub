package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/jayl2kor/skillhub/internal/skill"

	"github.com/spf13/cobra"
)

var (
	createType        string
	createAuthor      string
	createDescription string
)

var nameRegex = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

var entryFiles = map[string]string{
	"prompt": "prompt.md",
	"shell":  "script.sh",
	"python": "main.py",
	"node":   "index.js",
}

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Scaffold a new skill project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if !nameRegex.MatchString(name) {
			return fmt.Errorf("invalid skill name %q (must be lowercase alphanumeric with hyphens)", name)
		}

		validTypes := map[string]bool{"prompt": true, "shell": true, "python": true, "node": true}
		if !validTypes[createType] {
			return fmt.Errorf("unsupported type %q (supported: prompt, shell, python, node)", createType)
		}

		if _, err := os.Stat(name); err == nil {
			return fmt.Errorf("directory %q already exists", name)
		}

		if err := os.MkdirAll(name, 0755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}

		// Generate skill.json
		entry := entryFiles[createType]
		desc := createDescription
		if desc == "" {
			desc = "A new skillhub skill"
		}

		m := skill.Manifest{
			Name:        name,
			Version:     "0.1.0",
			Description: desc,
			Entry:       entry,
			Type:        createType,
		}
		if createAuthor != "" {
			m.Author = createAuthor
		}

		data, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling manifest: %w", err)
		}
		if err := os.WriteFile(filepath.Join(name, "skill.json"), append(data, '\n'), 0644); err != nil {
			return fmt.Errorf("writing skill.json: %w", err)
		}

		// Generate entry file
		entryContent := generateEntryContent(name, createType)
		if err := os.WriteFile(filepath.Join(name, entry), []byte(entryContent), 0644); err != nil {
			return fmt.Errorf("writing entry file: %w", err)
		}

		// Generate SKILL.md
		skillMD := fmt.Sprintf("# %s\n\n%s\n", name, desc)
		if err := os.WriteFile(filepath.Join(name, "SKILL.md"), []byte(skillMD), 0644); err != nil {
			return fmt.Errorf("writing SKILL.md: %w", err)
		}

		fmt.Printf("Created skill %q in ./%s/\n", name, name)
		return nil
	},
}

func init() {
	createCmd.Flags().StringVar(&createType, "type", "prompt", "skill type (prompt, shell, python, node)")
	createCmd.Flags().StringVar(&createAuthor, "author", "", "author name")
	createCmd.Flags().StringVar(&createDescription, "description", "", "skill description")
	rootCmd.AddCommand(createCmd)
}

func generateEntryContent(name, skillType string) string {
	switch skillType {
	case "prompt":
		return fmt.Sprintf("# %s\n\nDescribe your prompt here.\n", name)
	case "shell":
		return "#!/usr/bin/env bash\nset -euo pipefail\n\necho \"Hello from shell skill\"\n"
	case "python":
		return "#!/usr/bin/env python3\n\ndef main():\n    print(\"Hello from python skill\")\n\nif __name__ == \"__main__\":\n    main()\n"
	case "node":
		return "\"use strict\";\n\nconsole.log(\"Hello from node skill\");\n"
	default:
		return ""
	}
}
