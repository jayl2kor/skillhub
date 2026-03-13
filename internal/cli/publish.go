package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jayl2kor/skillhub/internal/installer"
	"github.com/jayl2kor/skillhub/internal/registry"
	"github.com/jayl2kor/skillhub/internal/skill"

	"github.com/spf13/cobra"
)

var (
	publishRepo    string
	publishVersion string
	publishForce   bool
	publishDryRun  bool
)

var publishCmd = &cobra.Command{
	Use:   "publish [dir]",
	Short: "Publish a skill to a registry",
	Long:  "Validate, package, and upload a skill to a configured registry.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}

		// Load and validate manifest
		manifestPath := filepath.Join(dir, "skill.json")
		m, err := skill.LoadManifest(manifestPath)
		if err != nil {
			return fmt.Errorf("loading manifest: %w", err)
		}
		if err := m.Validate(); err != nil {
			return fmt.Errorf("invalid manifest: %w", err)
		}

		// Verify entry file exists
		entryPath := filepath.Join(dir, m.Entry)
		if _, err := os.Stat(entryPath); os.IsNotExist(err) {
			return fmt.Errorf("entry file %q not found", m.Entry)
		}

		// Verify SKILL.md exists
		skillMDPath := filepath.Join(dir, "SKILL.md")
		if _, err := os.Stat(skillMDPath); os.IsNotExist(err) {
			return fmt.Errorf("SKILL.md not found (required)")
		}

		// Apply version override
		if publishVersion != "" {
			m.Version = publishVersion
			if err := m.Validate(); err != nil {
				return fmt.Errorf("invalid version override: %w", err)
			}
		}

		// Load config and resolve registry
		cfg, err := loadOrSetupConfig()
		if err != nil {
			return err
		}
		if len(cfg.Registries) == 0 {
			return fmt.Errorf("no registries configured; run 'skillhub repo add <url>' first")
		}

		var reg *registry.RepoSource
		if publishRepo != "" {
			for _, r := range cfg.Registries {
				if r.Name == publishRepo {
					reg = &registry.RepoSource{
						Name:     r.Name,
						URL:      r.URL,
						Token:    r.Token,
						Username: r.Username,
						Branch:   r.Branch,
					}
					break
				}
			}
			if reg == nil {
				return fmt.Errorf("registry %q not found in config", publishRepo)
			}
		} else if len(cfg.Registries) == 1 {
			r := cfg.Registries[0]
			reg = &registry.RepoSource{
				Name:     r.Name,
				URL:      r.URL,
				Token:    r.Token,
				Username: r.Username,
				Branch:   r.Branch,
			}
		} else {
			return fmt.Errorf("multiple registries configured; use --repo to specify")
		}

		// Build archive to temp directory
		if err := os.MkdirAll(paths.TmpDir, 0755); err != nil {
			return fmt.Errorf("creating temp directory: %w", err)
		}
		filename := fmt.Sprintf("%s-%s.tar.gz", m.Name, m.Version)
		archivePath := filepath.Join(paths.TmpDir, filename)
		defer os.Remove(archivePath)

		if err := createTarGz(dir, archivePath, m.Name); err != nil {
			return fmt.Errorf("creating archive: %w", err)
		}

		// Compute checksum
		checksum, err := installer.ComputeSHA256(archivePath)
		if err != nil {
			return fmt.Errorf("computing checksum: %w", err)
		}

		info, err := os.Stat(archivePath)
		if err != nil {
			return fmt.Errorf("stat archive: %w", err)
		}

		logVerbose("Archive: %s (%d bytes)", filename, info.Size())
		logVerbose("Checksum: %s", checksum)

		if publishDryRun {
			fmt.Printf("[dry-run] Would publish %s@%s to %s\n", m.Name, m.Version, reg.Name)
			fmt.Printf("  Archive: %s (%d bytes)\n", filename, info.Size())
			fmt.Printf("  Checksum: %s\n", checksum)
			return nil
		}

		// Upload
		client := registry.NewClient()

		if !reg.IsLocal() && (reg.Token == "" && reg.Username == "") {
			return fmt.Errorf("registry %q has no credentials; publishing to GitHub requires --token", reg.Name)
		}

		// Upload archive
		archiveContent, err := os.ReadFile(archivePath)
		if err != nil {
			return fmt.Errorf("reading archive: %w", err)
		}

		sha := ""
		if !reg.IsLocal() {
			sha, err = client.GetFileSHA(reg, filename)
			if err != nil {
				return fmt.Errorf("checking existing file: %w", err)
			}
			if sha != "" && !publishForce {
				return fmt.Errorf("archive %s already exists in %s (use --force to overwrite)", filename, reg.Name)
			}
		}

		commitMsg := fmt.Sprintf("publish %s@%s", m.Name, m.Version)
		if err := client.UploadFile(reg, filename, archiveContent, sha, commitMsg); err != nil {
			return fmt.Errorf("uploading archive: %w", err)
		}

		// Update index
		entry := registry.IndexEntry{
			Name:        m.Name,
			Version:     m.Version,
			Description: m.Description,
			Tags:        m.Tags,
			DownloadURL: filename,
			Checksum:    checksum,
		}
		if err := client.UpdateIndex(reg, entry, publishForce); err != nil {
			return fmt.Errorf("updating index: %w", err)
		}

		fmt.Printf("Published %s@%s to %s\n", m.Name, m.Version, reg.Name)
		return nil
	},
}

func init() {
	publishCmd.Flags().StringVar(&publishRepo, "repo", "", "target registry name")
	publishCmd.Flags().StringVar(&publishVersion, "version", "", "override version from skill.json")
	publishCmd.Flags().BoolVar(&publishForce, "force", false, "overwrite existing version")
	publishCmd.Flags().BoolVar(&publishDryRun, "dry-run", false, "validate and package without uploading")
	rootCmd.AddCommand(publishCmd)
}
