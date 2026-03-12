package cli

import (
	"fmt"
	"text/tabwriter"
	"os"

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
	rootCmd.AddCommand(listCmd)
}
