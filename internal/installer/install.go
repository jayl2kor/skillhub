// Package installer downloads, extracts, and verifies skill archives.
package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jayl2kor/skillhub/internal/config"
	"github.com/jayl2kor/skillhub/internal/registry"
	"github.com/jayl2kor/skillhub/internal/skill"
	"github.com/jayl2kor/skillhub/internal/storage"
)

type Installer struct {
	Paths      *storage.Paths
	Config     *config.Config
	Client     *registry.Client
	Verbose    func(format string, args ...any)
	AgentTool  string // agent type for --global install path (default: "claude")
	Version    string // specific version to install (empty = latest)
	RepoFilter string // filter by registry name (empty = all)
}

func NewInstaller(paths *storage.Paths, cfg *config.Config) *Installer {
	return &Installer{
		Paths:   paths,
		Config:  cfg,
		Client:  registry.NewClient(),
		Verbose: func(string, ...any) {},
	}
}

func (inst *Installer) logVerbose(format string, args ...any) {
	if inst.Verbose != nil {
		inst.Verbose(format, args...)
	}
}

func (inst *Installer) Install(name string, force bool, global bool) error {
	// 1. Check if already installed
	inst.logVerbose("checking if %q is already installed", name)
	if !force && storage.IsInstalled(inst.Paths, name) {
		return fmt.Errorf("skill %q is already installed (use --force to reinstall)", name)
	}

	// 2. Build registry sources (optionally filtered by RepoFilter)
	var entries []config.RegistryEntry
	if inst.RepoFilter != "" {
		for _, r := range inst.Config.Registries {
			if r.Name == inst.RepoFilter {
				entries = append(entries, r)
				break
			}
		}
		if len(entries) == 0 {
			return fmt.Errorf("registry %q not found; check 'skillhub repo list'", inst.RepoFilter)
		}
	} else {
		entries = inst.Config.Registries
	}

	if len(entries) == 0 {
		return fmt.Errorf("no registries configured; use 'skillhub repo add' to add one")
	}

	sources := make([]registry.RepoSource, len(entries))
	for i, r := range entries {
		sources[i] = registry.RepoSource{Name: r.Name, URL: r.URL, Token: r.Token, Username: r.Username, Branch: r.Branch}
	}

	// 3. Fetch and merge all indexes
	inst.logVerbose("fetching indexes from %d registry(ies)", len(sources))
	idx, err := inst.Client.FetchAllIndexes(sources)
	if err != nil {
		return fmt.Errorf("fetching indexes: %w", err)
	}
	inst.logVerbose("found %d skill(s) across registries", len(idx.Skills))

	// 4. Find skill
	var entry *registry.IndexEntry
	if inst.Version != "" {
		entry = idx.FindVersion(name, inst.Version)
		if entry == nil {
			return fmt.Errorf("skill %q version %q not found in any registry", name, inst.Version)
		}
	} else {
		entry = idx.Find(name)
		if entry == nil {
			return fmt.Errorf("skill %q not found in any registry", name)
		}
	}

	// 5. Resolve download URL and credentials
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

	// 6. Download to temp directory
	tmpDir, err := os.MkdirTemp(inst.Paths.TmpDir, name+"-*")
	if err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if strings.HasSuffix(entry.DownloadURL, "/") {
		// Directory mode: download files directly
		if matchedSource == nil {
			return fmt.Errorf("registry source not found for skill %q", name)
		}
		inst.logVerbose("downloading directory %s", entry.DownloadURL)
		inst.Client.OnProgress = func(filename string) {
			inst.logVerbose("  %s", filename)
		}
		if err := inst.Client.DownloadDirectory(matchedSource, entry.DownloadURL, tmpDir); err != nil {
			return fmt.Errorf("downloading skill directory: %w", err)
		}
	} else {
		// Archive mode: download tar.gz, verify, extract
		cacheFile := filepath.Join(inst.Paths.CacheDir, fmt.Sprintf("%s-%s.tar.gz", name, entry.Version))
		inst.logVerbose("downloading %s to %s", downloadURL, cacheFile)
		if err := inst.Client.Download(downloadURL, cacheFile, token, username); err != nil {
			return fmt.Errorf("downloading skill: %w", err)
		}

		if entry.Checksum != "" {
			if err := VerifyChecksum(cacheFile, entry.Checksum); err != nil {
				os.Remove(cacheFile)
				return fmt.Errorf("checksum verification failed: %w", err)
			}
		}

		inst.logVerbose("extracting archive to %s", tmpDir)
		if err := ExtractTarGz(cacheFile, tmpDir); err != nil {
			return fmt.Errorf("extracting archive: %w", err)
		}
	}

	// 9. Find and validate manifest (may be in root or subdirectory)
	manifestPath := findManifest(tmpDir)
	if manifestPath == "" {
		return fmt.Errorf("skill archive does not contain skill.json")
	}

	m, err := skill.LoadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	if err := m.Validate(); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	entryPath := filepath.Join(filepath.Dir(manifestPath), m.Entry)
	if _, err := os.Stat(entryPath); os.IsNotExist(err) {
		return fmt.Errorf("entry file %q not found in skill archive", m.Entry)
	}

	// 9.5. SKILL.md is required
	skillMDPath := filepath.Join(filepath.Dir(manifestPath), "SKILL.md")
	if _, err := os.Stat(skillMDPath); os.IsNotExist(err) {
		return fmt.Errorf("skill does not contain SKILL.md")
	}

	// 10. Determine install target
	var finalDir string
	if global {
		agentTool := inst.AgentTool
		if agentTool == "" {
			agentTool = "claude"
		}
		agent, err := skill.LookupAgent(agentTool)
		if err != nil {
			return err
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("detecting home directory: %w", err)
		}
		finalDir = filepath.Join(home, agent.SkillsPath, name)
	} else {
		finalDir = inst.Paths.SkillDir(name)
	}
	inst.logVerbose("installing to %s", finalDir)
	if err := os.MkdirAll(filepath.Dir(finalDir), 0755); err != nil {
		return fmt.Errorf("creating install directory: %w", err)
	}

	if force {
		if err := os.RemoveAll(finalDir); err != nil {
			return fmt.Errorf("removing existing skill for reinstall: %w", err)
		}
	}

	// The source is the directory containing skill.json
	sourceDir := filepath.Dir(manifestPath)
	if err := os.Rename(sourceDir, finalDir); err != nil {
		return fmt.Errorf("installing skill: %w", err)
	}

	// 11. Write install metadata
	meta := skill.NewInstallMeta(entry.Registry, entry.Version, entry.Checksum)
	metaPath := filepath.Join(finalDir, ".install.json")
	if err := meta.Save(metaPath); err != nil {
		return fmt.Errorf("writing install metadata: %w", err)
	}

	// 12. Success
	fmt.Printf("Installed %s@%s from %s\n", name, entry.Version, entry.Registry)
	return nil
}

func findManifest(dir string) string {
	// Check root
	root := filepath.Join(dir, "skill.json")
	if _, err := os.Stat(root); err == nil {
		return root
	}

	// Check one level of subdirectories
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			sub := filepath.Join(dir, entry.Name(), "skill.json")
			if _, err := os.Stat(sub); err == nil {
				return sub
			}
		}
	}

	return ""
}
