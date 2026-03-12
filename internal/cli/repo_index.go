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

	"github.com/spf13/cobra"
)

var (
	repoIndexMerge string
	repoIndexURL   string
)

var repoIndexCmd = &cobra.Command{
	Use:   "index <dir>",
	Short: "Generate index.json from packaged skills",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]

		// Find all .tar.gz files
		archives, err := filepath.Glob(filepath.Join(dir, "*.tar.gz"))
		if err != nil {
			return fmt.Errorf("scanning directory: %w", err)
		}
		if len(archives) == 0 {
			return fmt.Errorf("no .tar.gz files found in %s", dir)
		}

		var entries []registry.IndexEntry

		for _, archivePath := range archives {
			entry, err := indexArchive(archivePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[WARN] skipping %s: %v\n", filepath.Base(archivePath), err)
				continue
			}
			entries = append(entries, *entry)
		}

		if len(entries) == 0 {
			return fmt.Errorf("no valid skill archives found")
		}

		// Merge with existing index if requested
		if repoIndexMerge != "" {
			data, err := os.ReadFile(repoIndexMerge)
			if err != nil {
				return fmt.Errorf("reading existing index: %w", err)
			}
			existing, err := registry.ParseIndex(data)
			if err != nil {
				return fmt.Errorf("parsing existing index: %w", err)
			}

			// Add existing entries that aren't overridden by new ones
			newNames := make(map[string]bool)
			for _, e := range entries {
				newNames[e.Name] = true
			}
			for _, e := range existing.Skills {
				if !newNames[e.Name] {
					entries = append(entries, e)
				}
			}
		}

		idx := registry.Index{Skills: entries}
		data, err := json.MarshalIndent(idx, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling index: %w", err)
		}

		outputPath := filepath.Join(dir, "index.json")
		if err := os.WriteFile(outputPath, append(data, '\n'), 0644); err != nil {
			return fmt.Errorf("writing index: %w", err)
		}

		fmt.Printf("Generated %s with %d skill(s)\n", outputPath, len(entries))
		return nil
	},
}

func init() {
	repoIndexCmd.Flags().StringVar(&repoIndexMerge, "merge", "", "merge with existing index file")
	repoIndexCmd.Flags().StringVar(&repoIndexURL, "url", "", "base URL prefix for download_url fields")
	repoCmd.AddCommand(repoIndexCmd)
}

func indexArchive(archivePath string) (*registry.IndexEntry, error) {
	tmpDir, err := os.MkdirTemp("", "skillhub-index-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	if err := installer.ExtractTarGz(archivePath, tmpDir); err != nil {
		return nil, fmt.Errorf("extracting: %w", err)
	}

	// Find skill.json
	skillDir := findIndexSkillDir(tmpDir)
	if skillDir == "" {
		return nil, fmt.Errorf("no skill.json found")
	}

	m, err := skill.LoadManifest(filepath.Join(skillDir, "skill.json"))
	if err != nil {
		return nil, err
	}

	checksum, err := installer.ComputeSHA256(archivePath)
	if err != nil {
		return nil, fmt.Errorf("computing checksum: %w", err)
	}

	downloadURL := filepath.Base(archivePath)
	if repoIndexURL != "" {
		downloadURL = strings.TrimSuffix(repoIndexURL, "/") + "/" + filepath.Base(archivePath)
	}

	return &registry.IndexEntry{
		Name:        m.Name,
		Version:     m.Version,
		Description: m.Description,
		Tags:        m.Tags,
		DownloadURL: downloadURL,
		Checksum:    "sha256:" + checksum,
	}, nil
}

func findIndexSkillDir(dir string) string {
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
