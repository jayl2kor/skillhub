package cli

import (
	"github.com/jayl2kor/skillhub/internal/installer"

	"github.com/spf13/cobra"
)

var (
	forceInstall  bool
	globalInstall bool
)

var installCmd = &cobra.Command{
	Use:   "install <skill>",
	Short: "Install a skill",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadOrSetupConfig()
		if err != nil {
			return err
		}

		inst := installer.NewInstaller(paths, cfg)
		return inst.Install(args[0], forceInstall, globalInstall)
	},
}

func init() {
	installCmd.Flags().BoolVar(&forceInstall, "force", false, "force reinstall if already installed")
	installCmd.Flags().BoolVar(&globalInstall, "global", false, "install to project .claude/skills/ for Claude Code")
	rootCmd.AddCommand(installCmd)
}
