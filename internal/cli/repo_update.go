package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jayl2kor/skillhub/internal/registry"

	"github.com/spf13/cobra"
)

var repoUpdateCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Fetch and cache registry indexes locally",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadOrSetupConfig(cmd.Context())
		if err != nil {
			return err
		}

		if len(cfg.Registries) == 0 {
			return fmt.Errorf("no registries configured; use 'skillhub repo add' to add one")
		}

		indexDir := filepath.Join(paths.CacheDir, "indexes")
		if err := os.MkdirAll(indexDir, 0755); err != nil {
			return fmt.Errorf("creating index cache directory: %w", err)
		}

		client := registry.NewClient()
		ctx := cmd.Context()
		var updated int

		for _, r := range cfg.Registries {
			if len(args) > 0 && r.Name != args[0] {
				continue
			}

			src := &registry.RepoSource{
				Name: r.Name, URL: r.URL, Token: r.Token,
				Username: r.Username, Branch: r.Branch,
			}

			logVerbose("fetching index from %s", r.Name)
			idx, err := client.FetchIndex(ctx, src)
			if err != nil {
				fmt.Fprintf(os.Stderr, "WARNING: failed to update %q: %v\n", r.Name, err)
				continue
			}

			data, err := json.MarshalIndent(idx, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "WARNING: failed to marshal index for %q: %v\n", r.Name, err)
				continue
			}

			cachePath := filepath.Join(indexDir, r.Name+".json")
			if err := os.WriteFile(cachePath, data, 0644); err != nil {
				return fmt.Errorf("writing cached index for %q: %w", r.Name, err)
			}

			fmt.Printf("Updated %q (%d skills)\n", r.Name, len(idx.Skills))
			updated++
		}

		if len(args) > 0 && updated == 0 {
			return fmt.Errorf("registry %q not found", args[0])
		}

		return nil
	},
}

func init() {
	repoCmd.AddCommand(repoUpdateCmd)
}
