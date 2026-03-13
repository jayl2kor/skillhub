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
	repoIndexMerge  string
	repoIndexURL    string
	repoIndexOutput string
)

var repoIndexCmd = &cobra.Command{
	Use:   "index <dir>",
	Short: "Generate index.json from skill directories and archives",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]

		var entries []registry.IndexEntry
		seen := make(map[string]bool)

		// 1. Scan for skill directories (subdirectories containing skill.json)
		subdirs, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("scanning directory: %w", err)
		}
		for _, d := range subdirs {
			if !d.IsDir() {
				continue
			}
			if _, err := os.Stat(filepath.Join(dir, d.Name(), "skill.json")); err != nil {
				continue
			}
			entry, err := indexSkillDir(filepath.Join(dir, d.Name()))
			if err != nil {
				fmt.Fprintf(os.Stderr, "[WARN] skipping directory %s: %v\n", d.Name(), err)
				continue
			}
			entries = append(entries, *entry)
			seen[entry.Name] = true
		}

		// 2. Scan for .tar.gz archives (skip if directory already found for same skill)
		archives, _ := filepath.Glob(filepath.Join(dir, "*.tar.gz"))
		for _, archivePath := range archives {
			entry, err := indexArchive(archivePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[WARN] skipping %s: %v\n", filepath.Base(archivePath), err)
				continue
			}
			if seen[entry.Name] {
				continue
			}
			entries = append(entries, *entry)
			seen[entry.Name] = true
		}

		if len(entries) == 0 {
			return fmt.Errorf("no skill directories or archives found in %s", dir)
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

		outputPath := repoIndexOutput
		if outputPath == "" {
			outputPath = filepath.Join(dir, "index.json")
		}
		if err := os.WriteFile(outputPath, append(data, '\n'), 0644); err != nil {
			return fmt.Errorf("writing index: %w", err)
		}

		fmt.Printf("Generated %s with %d skill(s)\n", outputPath, len(entries))
		return nil
	},
}

func init() {
	repoIndexCmd.Flags().StringVarP(&repoIndexMerge, "merge", "m", "", "merge with existing index file")
	repoIndexCmd.Flags().StringVarP(&repoIndexURL, "url", "u", "", "base URL prefix for download_url fields")
	repoIndexCmd.Flags().StringVarP(&repoIndexOutput, "output", "o", "", "output path for index.json (default: <dir>/index.json)")
	repoCmd.AddCommand(repoIndexCmd)
}

func indexSkillDir(dirPath string) (*registry.IndexEntry, error) {
	m, err := skill.LoadManifest(filepath.Join(dirPath, "skill.json"))
	if err != nil {
		return nil, err
	}

	dirName := filepath.Base(dirPath)
	downloadURL := dirName + "/"
	if repoIndexURL != "" {
		downloadURL = strings.TrimSuffix(repoIndexURL, "/") + "/" + dirName + "/"
	}

	return &registry.IndexEntry{
		Name:        m.Name,
		Version:     m.Version,
		Description: m.Description,
		Tags:        m.Tags,
		DownloadURL: downloadURL,
	}, nil
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
