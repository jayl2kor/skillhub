package cli

import (
	"fmt"
	"os"

	"github.com/jayl2kor/skillhub/internal/storage"
	"github.com/jayl2kor/skillhub/pkg/version"

	"github.com/spf13/cobra"
)

var (
	homeDir string
	verbose bool
	paths   *storage.Paths
)

var rootCmd = &cobra.Command{
	Use:     "skillhub",
	Short:   "A lightweight package manager for AI/agent skills",
	Long:    "skillhub manages the discovery, installation, execution, and lifecycle of reusable AI/agent skills.",
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version.Version, version.GitCommit, version.BuildDate),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if homeDir == "" {
			homeDir = storage.DefaultHome()
		}
		paths = storage.NewPaths(homeDir)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&homeDir, "home", "", "skillhub home directory (default: ~/.skillhub)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
}

func Execute() error {
	return rootCmd.Execute()
}

func logVerbose(format string, args ...any) {
	if verbose {
		fmt.Fprintf(os.Stderr, "[verbose] "+format+"\n", args...)
	}
}
