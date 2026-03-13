package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/jayl2kor/skillhub/internal/config"

	"github.com/spf13/cobra"
)

var configUnmask bool

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and modify skillhub configuration",
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all configuration values",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		kvs := cfg.ListValues()

		if isStructuredOutput() {
			m := make(map[string]string, len(kvs))
			for _, kv := range kvs {
				v := kv.Value
				if !configUnmask && isTokenKey(kv.Key) {
					v = maskToken(v)
				}
				m[kv.Key] = v
			}
			return printFormatted(m)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "KEY\tVALUE")
		for _, kv := range kvs {
			v := kv.Value
			if !configUnmask && isTokenKey(kv.Key) {
				v = maskToken(v)
			}
			fmt.Fprintf(w, "%s\t%s\n", kv.Key, v)
		}
		return w.Flush()
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		val, err := cfg.GetValue(args[0])
		if err != nil {
			return err
		}

		if !configUnmask && isTokenKey(args[0]) {
			val = maskToken(val)
		}

		if isStructuredOutput() {
			return printFormatted(map[string]string{"key": args[0], "value": val})
		}

		fmt.Println(val)
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		if err := cfg.SetValue(args[0], args[1]); err != nil {
			return err
		}

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}

		if err := cfg.Save(paths.Config); err != nil {
			return err
		}

		fmt.Printf("Set %s = %s\n", args[0], args[1])
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(paths.Config)
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open configuration in $EDITOR",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(paths.Config); os.IsNotExist(err) {
			return fmt.Errorf("config not found at %s; run 'skillhub init' to create it", paths.Config)
		}

		editor := os.Getenv("EDITOR")
		if editor == "" {
			for _, fallback := range []string{"vi", "nano"} {
				if _, err := exec.LookPath(fallback); err == nil {
					editor = fallback
					break
				}
			}
		}
		if editor == "" {
			return fmt.Errorf("no editor found; set $EDITOR environment variable")
		}

		c := exec.Command(editor, paths.Config)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("editor exited with error: %w", err)
		}

		// Validate after edit
		cfg, err := config.Load(paths.Config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: config may be invalid: %v\n", err)
			return nil
		}
		if err := cfg.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: config validation failed: %v\n", err)
		}
		return nil
	},
}

func init() {
	configListCmd.Flags().BoolVar(&configUnmask, "unmask", false, "show tokens without masking")
	configListCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format (table, json, yaml)")
	configGetCmd.Flags().BoolVar(&configUnmask, "unmask", false, "show tokens without masking")
	configGetCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format (table, json, yaml)")
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configEditCmd)
	rootCmd.AddCommand(configCmd)
}

// loadConfig loads config without triggering interactive setup.
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load(paths.Config)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config not found at %s; run 'skillhub init' to create it", paths.Config)
		}
		return nil, err
	}
	return cfg, nil
}

func maskToken(s string) string {
	if s == "" {
		return ""
	}
	if len(s) >= 8 {
		return "****" + s[len(s)-4:]
	}
	return "********"
}

func isTokenKey(key string) bool {
	return strings.HasSuffix(key, ".token") || key == "token"
}
