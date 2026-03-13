package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jayl2kor/skillhub/internal/registry"
	"github.com/jayl2kor/skillhub/internal/skill"

	"github.com/spf13/cobra"
)

var (
	publishRepo    string
	publishVersion string
	publishToken   string
	publishForce   bool
	publishDryRun  bool
	publishPrefix  string
)

var publishCmd = &cobra.Command{
	Use:   "publish [dir]",
	Short: "Publish a skill to a registry",
	Long:  "Validate and upload a skill directory to a configured registry.",
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
		var configSkillsPrefix string
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
					configSkillsPrefix = r.SkillsPrefix
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
			configSkillsPrefix = r.SkillsPrefix
		} else {
			return fmt.Errorf("multiple registries configured; use --repo to specify")
		}

		// --token flag overrides config token (useful for public registries
		// that were added without a token but need write access for publish)
		if publishToken != "" {
			reg.Token = publishToken
		}

		// Resolve skills prefix: --prefix flag > per-registry config > default
		skillsPrefix := ".claude/skills"
		if configSkillsPrefix != "" {
			skillsPrefix = configSkillsPrefix
		}
		if publishPrefix != "" {
			skillsPrefix = publishPrefix
		}
		destPrefix := skillsPrefix + "/" + m.Name

		if publishDryRun {
			fmt.Printf("[dry-run] Would publish %s@%s to %s (%s/)\n", m.Name, m.Version, reg.Name, destPrefix)
			return nil
		}

		client := registry.NewClient()

		if !reg.IsLocal() && reg.Token == "" && reg.Username == "" {
			return fmt.Errorf("registry %q has no credentials for write access; use --token <PAT> or configure via 'skillhub repo add <url> --token <PAT>'", reg.Name)
		}

		// Check version conflict via index
		if !publishForce {
			idx, err := client.FetchIndex(reg)
			if err == nil {
				if existing := idx.FindVersion(m.Name, m.Version); existing != nil {
					return fmt.Errorf("version %s@%s already exists in %s (use --force to overwrite)", m.Name, m.Version, reg.Name)
				}
			}
		}

		// Upload skill directory
		commitMsg := fmt.Sprintf("publish %s@%s", m.Name, m.Version)
		if verbose {
			client.OnProgress = func(filename string) {
				logVerbose("uploaded %s", filename)
			}
		}

		if err := client.UploadDirectory(reg, dir, destPrefix, commitMsg); err != nil {
			return fmt.Errorf("uploading skill directory: %w", err)
		}

		// Update index
		entry := registry.IndexEntry{
			Name:        m.Name,
			Version:     m.Version,
			Description: m.Description,
			Tags:        m.Tags,
			DownloadURL: destPrefix + "/",
		}
		if err := client.UpdateIndex(reg, entry, publishForce); err != nil {
			return fmt.Errorf("updating index: %w", err)
		}

		fmt.Printf("Published %s@%s to %s\n", m.Name, m.Version, reg.Name)
		return nil
	},
}

func init() {
	publishCmd.Flags().StringVarP(&publishRepo, "repo", "r", "", "target registry name")
	publishCmd.Flags().StringVar(&publishVersion, "version", "", "override version from skill.json")
	publishCmd.Flags().StringVar(&publishToken, "token", "", "GitHub personal access token (overrides config)")
	publishCmd.Flags().BoolVarP(&publishForce, "force", "f", false, "overwrite existing version")
	publishCmd.Flags().BoolVarP(&publishDryRun, "dry-run", "n", false, "validate without uploading")
	publishCmd.Flags().StringVar(&publishPrefix, "prefix", "", "destination prefix in registry (default: .claude/skills)")
	rootCmd.AddCommand(publishCmd)
}
