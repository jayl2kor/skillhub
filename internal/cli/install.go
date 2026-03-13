package cli

import (
	"github.com/jayl2kor/skillhub/internal/installer"

	"github.com/spf13/cobra"
)

var (
	forceInstall   bool
	globalInstall  bool
	installTool    string
	installVersion string
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
		inst.Verbose = logVerbose
		inst.AgentTool = installTool
		inst.Version = installVersion
		return inst.Install(args[0], forceInstall, globalInstall)
	},
}

func init() {
	installCmd.Flags().BoolVarP(&forceInstall, "force", "f", false, "force reinstall if already installed")
	installCmd.Flags().BoolVarP(&globalInstall, "global", "g", false, "install to agent skills directory in home")
	installCmd.Flags().StringVarP(&installTool, "tool", "t", "claude", "agent type for --global install path (claude, cursor, windsurf, cline, generic)")
	installCmd.Flags().StringVar(&installVersion, "version", "", "install a specific version")
	rootCmd.AddCommand(installCmd)
}
