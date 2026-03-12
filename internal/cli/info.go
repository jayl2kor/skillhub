package cli

import (
	"fmt"
	"strings"

	"github.com/jayl2kor/skillhub/internal/registry"
	"github.com/jayl2kor/skillhub/internal/storage"

	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info <skill>",
	Short: "Show detailed skill information",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Check if installed locally
		installed := storage.IsInstalled(paths, name)
		if installed {
			s, err := storage.GetInstalledSkill(paths, name)
			if err == nil {
				if isStructuredOutput() {
					return printFormatted(s)
				}
				fmt.Printf("Name:        %s\n", s.Manifest.Name)
				fmt.Printf("Version:     %s\n", s.Manifest.Version)
				fmt.Printf("Description: %s\n", s.Manifest.Description)
				fmt.Printf("Type:        %s\n", s.Manifest.Type)
				fmt.Printf("Entry:       %s\n", s.Manifest.Entry)
				if len(s.Manifest.Tags) > 0 {
					fmt.Printf("Tags:        %s\n", strings.Join(s.Manifest.Tags, ", "))
				}
				if s.Manifest.Author != "" {
					fmt.Printf("Author:      %s\n", s.Manifest.Author)
				}
				fmt.Printf("Installed:   yes\n")
				if s.Meta.Registry != "" {
					fmt.Printf("Registry:    %s\n", s.Meta.Registry)
				}
				if s.Meta.InstalledAt != "" {
					fmt.Printf("Installed at: %s\n", s.Meta.InstalledAt)
				}
				return nil
			}
		}

		// Try from registry
		cfg, err := loadOrSetupConfig()
		if err != nil {
			return err
		}

		if len(cfg.Registries) == 0 {
			if installed {
				return nil
			}
			return fmt.Errorf("skill %q not found locally and no registries configured", name)
		}

		sources := make([]registry.RepoSource, len(cfg.Registries))
		for i, r := range cfg.Registries {
			sources[i] = registry.RepoSource{Name: r.Name, URL: r.URL, Token: r.Token, Username: r.Username, Branch: r.Branch}
		}

		client := registry.NewClient()
		idx, err := client.FetchAllIndexes(sources)
		if err != nil {
			return fmt.Errorf("fetching indexes: %w", err)
		}

		entry := idx.Find(name)
		if entry == nil {
			return fmt.Errorf("skill %q not found", name)
		}

		if isStructuredOutput() {
			return printFormatted(entry)
		}

		fmt.Printf("Name:        %s\n", entry.Name)
		fmt.Printf("Version:     %s\n", entry.Version)
		fmt.Printf("Description: %s\n", entry.Description)
		if len(entry.Tags) > 0 {
			fmt.Printf("Tags:        %s\n", strings.Join(entry.Tags, ", "))
		}
		fmt.Printf("Registry:    %s\n", entry.Registry)
		fmt.Printf("Installed:   no\n")

		return nil
	},
}

func init() {
	infoCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format (table, json, yaml)")
	rootCmd.AddCommand(infoCmd)
}
