package cli

import (
	"fmt"
	"os"

	"github.com/jayl2kor/skillhub/internal/config"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the skillhub workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create directories
		if err := paths.EnsureDirectories(); err != nil {
			return fmt.Errorf("creating directories: %w", err)
		}
		fmt.Println("Created workspace directories at", paths.Home)

		// Write default config if not exists
		if _, err := os.Stat(paths.Config); os.IsNotExist(err) {
			cfg := config.DefaultConfig(paths.Home)
			if err := cfg.Save(paths.Config); err != nil {
				return fmt.Errorf("writing config: %w", err)
			}
			fmt.Println("Created default config at", paths.Config)
		} else {
			fmt.Println("Config already exists at", paths.Config)
		}

		fmt.Println("Workspace initialized successfully.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
