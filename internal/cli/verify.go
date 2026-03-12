package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jayl2kor/skillhub/internal/installer"
	"github.com/jayl2kor/skillhub/internal/skill"

	"github.com/spf13/cobra"
)

var verifyChecksum string

var verifyCmd = &cobra.Command{
	Use:   "verify <archive>",
	Short: "Verify a skill archive",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		archivePath := args[0]

		// Check archive exists
		if _, err := os.Stat(archivePath); os.IsNotExist(err) {
			return fmt.Errorf("archive not found: %s", archivePath)
		}

		// Verify checksum if provided
		if verifyChecksum != "" {
			if err := installer.VerifyChecksum(archivePath, verifyChecksum); err != nil {
				return fmt.Errorf("checksum verification failed: %w", err)
			}
			fmt.Println("Checksum OK")
		}

		// Extract to temp dir
		tmpDir, err := os.MkdirTemp("", "skillhub-verify-*")
		if err != nil {
			return fmt.Errorf("creating temp directory: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		if err := installer.ExtractTarGz(archivePath, tmpDir); err != nil {
			return fmt.Errorf("extracting archive: %w", err)
		}

		// Find skill.json
		skillDir := findVerifySkillDir(tmpDir)
		if skillDir == "" {
			return fmt.Errorf("archive does not contain skill.json")
		}

		// Load and validate manifest
		m, err := skill.LoadManifest(filepath.Join(skillDir, "skill.json"))
		if err != nil {
			return fmt.Errorf("loading manifest: %w", err)
		}
		if err := m.Validate(); err != nil {
			return fmt.Errorf("invalid manifest: %w", err)
		}

		// Check entry file exists
		entryPath := filepath.Join(skillDir, m.Entry)
		if _, err := os.Stat(entryPath); os.IsNotExist(err) {
			return fmt.Errorf("entry file %q not found in archive", m.Entry)
		}

		fmt.Printf("Verified %s@%s (%s)\n", m.Name, m.Version, m.Type)
		return nil
	},
}

func init() {
	verifyCmd.Flags().StringVar(&verifyChecksum, "checksum", "", "expected checksum (sha256:...)")
	rootCmd.AddCommand(verifyCmd)
}

func findVerifySkillDir(dir string) string {
	if _, err := os.Stat(filepath.Join(dir, "skill.json")); err == nil {
		return dir
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			sub := filepath.Join(dir, entry.Name())
			if _, err := os.Stat(filepath.Join(sub, "skill.json")); err == nil {
				return sub
			}
		}
	}
	return ""
}
