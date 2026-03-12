package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/jayl2kor/skillhub/internal/registry"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for skills in registries (omit query to list all)",
	Args:  cobra.MaximumNArgs(1),
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
		logVerbose("fetching indexes from %d registry(ies)", len(sources))
		idx, err := client.FetchAllIndexes(sources)
		if err != nil {
			return fmt.Errorf("fetching indexes: %w", err)
		}
		logVerbose("found %d skill(s) total", len(idx.Skills))

		var results []registry.IndexEntry
		if len(args) > 0 {
			results = idx.Search(args[0])
		} else {
			results = idx.Skills
		}
		if len(results) == 0 {
			fmt.Println("No skills found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tVERSION\tDESCRIPTION")
		for _, r := range results {
			desc := r.Description
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", r.Name, r.Version, desc)
		}
		w.Flush()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
