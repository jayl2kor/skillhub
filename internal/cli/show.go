package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jayl2kor/skillhub/internal/installer"
	"github.com/jayl2kor/skillhub/internal/registry"
	"github.com/jayl2kor/skillhub/internal/skill"
	"github.com/jayl2kor/skillhub/internal/storage"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show skill contents before installing",
	Long:  "Inspect skill files (manifest, readme, entry) for installed or remote skills.",
}

var showManifestCmd = &cobra.Command{
	Use:   "manifest <skill>",
	Short: "Show raw skill.json",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, cleanup, err := resolveSkillDir(args[0])
		if err != nil {
			return err
		}
		defer cleanup()

		data, err := os.ReadFile(filepath.Join(dir, "skill.json"))
		if err != nil {
			return fmt.Errorf("reading skill.json: %w", err)
		}
		fmt.Print(string(data))
		return nil
	},
}

var showReadmeCmd = &cobra.Command{
	Use:   "readme <skill>",
	Short: "Show SKILL.md",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, cleanup, err := resolveSkillDir(args[0])
		if err != nil {
			return err
		}
		defer cleanup()

		data, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
		if err != nil {
			return fmt.Errorf("reading SKILL.md: %w", err)
		}
		fmt.Print(string(data))
		return nil
	},
}

var showEntryCmd = &cobra.Command{
	Use:   "entry <skill>",
	Short: "Show entry file content",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, cleanup, err := resolveSkillDir(args[0])
		if err != nil {
			return err
		}
		defer cleanup()

		m, err := skill.LoadManifest(filepath.Join(dir, "skill.json"))
		if err != nil {
			return fmt.Errorf("loading manifest: %w", err)
		}

		data, err := os.ReadFile(filepath.Join(dir, m.Entry))
		if err != nil {
			return fmt.Errorf("reading entry file %q: %w", m.Entry, err)
		}
		fmt.Print(string(data))
		return nil
	},
}

var showAllCmd = &cobra.Command{
	Use:   "all <skill>",
	Short: "Show manifest and readme",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, cleanup, err := resolveSkillDir(args[0])
		if err != nil {
			return err
		}
		defer cleanup()

		manifest, err := os.ReadFile(filepath.Join(dir, "skill.json"))
		if err != nil {
			return fmt.Errorf("reading skill.json: %w", err)
		}

		// Pretty-print manifest JSON
		var raw json.RawMessage
		if err := json.Unmarshal(manifest, &raw); err == nil {
			if pretty, err := json.MarshalIndent(raw, "", "  "); err == nil {
				manifest = pretty
			}
		}

		fmt.Println(string(manifest))
		fmt.Println("---")

		readme, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
		if err != nil {
			fmt.Println("(no SKILL.md)")
			return nil
		}
		fmt.Print(string(readme))
		return nil
	},
}

func init() {
	showCmd.AddCommand(showManifestCmd)
	showCmd.AddCommand(showReadmeCmd)
	showCmd.AddCommand(showEntryCmd)
	showCmd.AddCommand(showAllCmd)
	rootCmd.AddCommand(showCmd)
}

// resolveSkillDir returns the directory containing the skill files.
// For installed skills, it returns the installed directory.
// For remote skills, it downloads and extracts to a temp directory.
// The caller must call cleanup() when done.
func resolveSkillDir(name string) (dir string, cleanup func(), err error) {
	noop := func() {}

	// 1. Check if installed locally
	if storage.IsInstalled(paths, name) {
		s, err := storage.GetInstalledSkill(paths, name)
		if err == nil {
			return s.Dir, noop, nil
		}
	}

	// 2. Fetch from registry
	cfg, err := loadOrSetupConfig()
	if err != nil {
		return "", noop, err
	}

	if len(cfg.Registries) == 0 {
		return "", noop, fmt.Errorf("skill %q not found locally and no registries configured", name)
	}

	sources := make([]registry.RepoSource, len(cfg.Registries))
	for i, r := range cfg.Registries {
		sources[i] = registry.RepoSource{Name: r.Name, URL: r.URL, Token: r.Token, Username: r.Username, Branch: r.Branch}
	}

	client := registry.NewClient()
	idx, err := client.FetchAllIndexes(sources)
	if err != nil {
		return "", noop, fmt.Errorf("fetching indexes: %w", err)
	}

	entry := idx.Find(name)
	if entry == nil {
		return "", noop, fmt.Errorf("skill %q not found", name)
	}

	// Resolve download URL and credentials
	var matchedSource *registry.RepoSource
	var downloadURL string
	var token, username string
	for i := range sources {
		if sources[i].Name == entry.Registry {
			matchedSource = &sources[i]
			downloadURL = sources[i].ResolveDownloadURL(entry.DownloadURL)
			token = sources[i].Token
			username = sources[i].Username
			break
		}
	}
	if downloadURL == "" {
		downloadURL = entry.DownloadURL
	}

	// Download to temp
	tmpDir, err := os.MkdirTemp("", "skillhub-show-*")
	if err != nil {
		return "", noop, fmt.Errorf("creating temp directory: %w", err)
	}
	cleanupFn := func() { os.RemoveAll(tmpDir) }

	extractDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		cleanupFn()
		return "", noop, fmt.Errorf("creating extract directory: %w", err)
	}

	if strings.HasSuffix(entry.DownloadURL, "/") {
		// Directory mode
		if matchedSource == nil {
			cleanupFn()
			return "", noop, fmt.Errorf("registry source not found for skill %q", name)
		}
		logVerbose("downloading directory %s", entry.DownloadURL)
		if err := client.DownloadDirectory(matchedSource, entry.DownloadURL, extractDir); err != nil {
			cleanupFn()
			return "", noop, fmt.Errorf("downloading skill directory: %w", err)
		}
	} else {
		// Archive mode
		archivePath := filepath.Join(tmpDir, "archive.tar.gz")
		logVerbose("downloading %s", downloadURL)
		if err := client.Download(downloadURL, archivePath, token, username); err != nil {
			cleanupFn()
			return "", noop, fmt.Errorf("downloading skill: %w", err)
		}

		if err := installer.ExtractTarGz(archivePath, extractDir); err != nil {
			cleanupFn()
			return "", noop, fmt.Errorf("extracting archive: %w", err)
		}
	}

	// Find manifest (same logic as installer.findManifest)
	skillDir := findSkillDir(extractDir)
	if skillDir == "" {
		cleanupFn()
		return "", noop, fmt.Errorf("skill archive does not contain skill.json")
	}

	return skillDir, cleanupFn, nil
}

// findSkillDir locates the directory containing skill.json within the extracted archive.
func findSkillDir(dir string) string {
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
