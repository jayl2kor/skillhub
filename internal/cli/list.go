package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/jayl2kor/skillhub/internal/storage"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		skills, err := storage.ListInstalledSkills(paths)
		if err != nil {
			return fmt.Errorf("listing skills: %w", err)
		}

		if isStructuredOutput() {
			type entry struct {
				Name    string `json:"name" yaml:"name"`
				Version string `json:"version" yaml:"version"`
				Type    string `json:"type" yaml:"type"`
				Dir     string `json:"dir" yaml:"dir"`
			}
			out := make([]entry, len(skills))
			for i, s := range skills {
				out[i] = entry{s.Manifest.Name, s.Manifest.Version, s.Manifest.Type, s.Dir}
			}
			return printFormatted(out)
		}

		if len(skills) == 0 {
			fmt.Println("No skills installed.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tVERSION\tTYPE")
		for _, s := range skills {
			fmt.Fprintf(w, "%s\t%s\t%s\n", s.Manifest.Name, s.Manifest.Version, s.Manifest.Type)
		}
		w.Flush()

		return nil
	},
}

func init() {
	listCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format (table, json, yaml)")
	rootCmd.AddCommand(listCmd)
}
