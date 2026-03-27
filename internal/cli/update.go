package cli

import (
	"fmt"

	"github.com/jayl2kor/skillhub/internal/installer"
	"github.com/jayl2kor/skillhub/internal/registry"
	"github.com/jayl2kor/skillhub/internal/skill"
	"github.com/jayl2kor/skillhub/internal/storage"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update [skill]",
	Short: "Update installed skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadOrSetupConfig()
		if err != nil {
			return err
		}

		if len(cfg.Registries) == 0 {
			return fmt.Errorf("no registries configured; use 'skillhub repo add' to add one")
		}

		sources := make([]registry.RepoSource, len(cfg.Registries))
		for i, r := range cfg.Registries {
			sources[i] = registry.RepoSource{Name: r.Name, URL: r.URL, Token: r.Token, Username: r.Username, Branch: r.Branch}
		}

		client := registry.NewClient()
		ctx := cmd.Context()
		idx, err := client.FetchAllIndexes(ctx, sources)
		if err != nil {
			return fmt.Errorf("fetching indexes: %w", err)
		}

		var skillsToUpdate []string

		if len(args) > 0 {
			// Update specific skill
			skillsToUpdate = args
		} else {
			// Update all installed skills
			installed, err := storage.ListInstalledSkills(paths)
			if err != nil {
				return fmt.Errorf("listing installed skills: %w", err)
			}
			for _, s := range installed {
				skillsToUpdate = append(skillsToUpdate, s.Manifest.Name)
			}
		}

		if len(skillsToUpdate) == 0 {
			fmt.Println("No skills to update.")
			return nil
		}

		inst := installer.NewInstaller(paths, cfg)
		updated := 0

		for _, name := range skillsToUpdate {
			entry := idx.Find(name)
			if entry == nil {
				fmt.Printf("%s: not found in registries\n", name)
				continue
			}

			// Check if installed
			s, err := storage.GetInstalledSkill(paths, name)
			if err != nil {
				fmt.Printf("%s: not installed locally\n", name)
				continue
			}

			// Compare versions
			cmp, err := skill.CompareVersions(s.Manifest.Version, entry.Version)
			if err != nil {
				fmt.Printf("%s: version comparison error: %v\n", name, err)
				continue
			}

			if cmp >= 0 {
				fmt.Printf("%s: already at latest version (%s)\n", name, s.Manifest.Version)
				continue
			}

			fmt.Printf("%s: updating %s -> %s\n", name, s.Manifest.Version, entry.Version)
			if err := inst.Install(ctx, name, true, false); err != nil {
				fmt.Printf("%s: update failed: %v\n", name, err)
				continue
			}
			updated++
		}

		if updated == 0 {
			fmt.Println("All skills are up to date.")
		} else {
			fmt.Printf("Updated %d skill(s).\n", updated)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
