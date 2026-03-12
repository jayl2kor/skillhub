package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/jayl2kor/skillhub/internal/registry"

	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage skill registries",
}

var repoAddCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Add a registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadOrSetupConfig()
		if err != nil {
			return err
		}

		source, err := registry.ParseRepoURL(args[0])
		if err != nil {
			return fmt.Errorf("parsing URL: %w", err)
		}

		token, _ := cmd.Flags().GetString("token")
		username, _ := cmd.Flags().GetString("username")
		source.Token = token
		source.Username = username

		// Detect default branch
		client := registry.NewClient()
		source.Branch = client.DetectDefaultBranch(source)

		// Verify index is accessible
		if _, err := client.FetchIndex(source); err != nil {
			return fmt.Errorf("cannot access registry index: %w", err)
		}

		if err := cfg.AddRegistry(source.Name, source.URL, source.Token, source.Username, source.Branch); err != nil {
			return err
		}

		if err := cfg.Save(paths.Config); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Printf("Added registry %q (%s)\n", source.Name, source.URL)
		return nil
	},
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered registries",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadOrSetupConfig()
		if err != nil {
			return err
		}

		if len(cfg.Registries) == 0 {
			fmt.Println("No registries configured.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tURL")
		for _, r := range cfg.Registries {
			fmt.Fprintf(w, "%s\t%s\n", r.Name, r.URL)
		}
		w.Flush()

		return nil
	},
}

var repoRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadOrSetupConfig()
		if err != nil {
			return err
		}

		if err := cfg.RemoveRegistry(args[0]); err != nil {
			return err
		}

		if err := cfg.Save(paths.Config); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Printf("Removed registry %q\n", args[0])
		return nil
	},
}

func init() {
	repoAddCmd.Flags().String("token", "", "personal access token for private registries")
	repoAddCmd.Flags().String("username", "", "username for Basic Auth (required for GitHub Enterprise)")
	repoCmd.AddCommand(repoAddCmd)
	repoCmd.AddCommand(repoListCmd)
	repoCmd.AddCommand(repoRemoveCmd)
	rootCmd.AddCommand(repoCmd)
}
