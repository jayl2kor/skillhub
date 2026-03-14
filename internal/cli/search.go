package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/jayl2kor/skillhub/internal/registry"

	"github.com/spf13/cobra"
)

var (
	searchVersions bool
	searchRepo     string
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

		sources, err := registrySources(cfg, searchRepo)
		if err != nil {
			return err
		}

		client := registry.NewClient()
		ctx := cmd.Context()
		logVerbose("fetching indexes from %d registry(ies)", len(sources))
		idx, err := client.FetchAllIndexes(ctx, sources)
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
		// Deduplicate by name unless --versions is set
		if !searchVersions {
			seen := make(map[string]bool)
			var deduped []registry.IndexEntry
			for _, r := range results {
				if !seen[r.Name] {
					seen[r.Name] = true
					deduped = append(deduped, r)
				}
			}
			results = deduped
		}

		if isStructuredOutput() {
			return printFormatted(results)
		}

		if len(results) == 0 {
			fmt.Println("No skills found.")
			return nil
		}

		multiRegistry := len(cfg.Registries) > 1

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		if multiRegistry {
			fmt.Fprintln(w, "NAME\tVERSION\tREGISTRY\tDESCRIPTION")
		} else {
			fmt.Fprintln(w, "NAME\tVERSION\tDESCRIPTION")
		}
		for _, r := range results {
			desc := r.Description
			runes := []rune(desc)
			if len(runes) > 60 {
				desc = string(runes[:57]) + "..."
			}
			if multiRegistry {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Name, r.Version, r.Registry, desc)
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\n", r.Name, r.Version, desc)
			}
		}
		w.Flush()

		return nil
	},
}

func init() {
	searchCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format (table, json, yaml)")
	searchCmd.Flags().BoolVarP(&searchVersions, "versions", "V", false, "show all available versions")
	searchCmd.Flags().StringVarP(&searchRepo, "repo", "r", "", "filter by registry name")
	rootCmd.AddCommand(searchCmd)
}
