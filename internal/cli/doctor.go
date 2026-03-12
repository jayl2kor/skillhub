package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jayl2kor/skillhub/internal/config"
	"github.com/jayl2kor/skillhub/internal/storage"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check local setup health",
	RunE: func(cmd *cobra.Command, args []string) error {
		ok := true

		// Check config
		fmt.Print("Config file... ")
		if _, err := os.Stat(paths.Config); err != nil {
			fmt.Println("MISSING")
			ok = false
		} else {
			cfg, err := config.Load(paths.Config)
			if err != nil {
				fmt.Printf("INVALID (%v)\n", err)
				ok = false
			} else if err := cfg.Validate(); err != nil {
				fmt.Printf("INVALID (%v)\n", err)
				ok = false
			} else {
				fmt.Println("OK")
			}
		}

		// Check directories
		for name, dir := range map[string]string{
			"Skills directory": paths.SkillsDir,
			"Cache directory":  paths.CacheDir,
			"Log directory":    paths.LogDir,
		} {
			fmt.Printf("%s... ", name)
			info, err := os.Stat(dir)
			if err != nil {
				fmt.Println("MISSING")
				ok = false
			} else if !info.IsDir() {
				fmt.Println("NOT A DIRECTORY")
				ok = false
			} else {
				fmt.Println("OK")
			}
		}

		// Check cache writable
		fmt.Print("Cache writable... ")
		testFile := filepath.Join(paths.CacheDir, ".doctor-test")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			fmt.Printf("NO (%v)\n", err)
			ok = false
		} else {
			os.Remove(testFile)
			fmt.Println("OK")
		}

		// Check installed skill manifests
		fmt.Print("Installed skills... ")
		skills, err := storage.ListInstalledSkills(paths)
		if err != nil {
			fmt.Printf("ERROR (%v)\n", err)
			ok = false
		} else {
			invalid := 0
			for _, s := range skills {
				if err := s.Manifest.Validate(); err != nil {
					invalid++
				}
			}
			if invalid > 0 {
				fmt.Printf("%d installed, %d with invalid manifests\n", len(skills), invalid)
				ok = false
			} else {
				fmt.Printf("%d installed, all valid\n", len(skills))
			}
		}

		// Check registries
		fmt.Print("Registries... ")
		cfg, err := config.Load(paths.Config)
		if err == nil && len(cfg.Registries) > 0 {
			fmt.Printf("%d configured\n", len(cfg.Registries))
		} else if err == nil {
			fmt.Println("none configured")
		}

		if !ok {
			return fmt.Errorf("some checks failed; run 'skillhub init' to fix")
		}

		fmt.Println("\nAll checks passed.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
