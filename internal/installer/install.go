package installer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jayl2kor/skillhub/internal/config"
	"github.com/jayl2kor/skillhub/internal/registry"
	"github.com/jayl2kor/skillhub/internal/skill"
	"github.com/jayl2kor/skillhub/internal/storage"
)

type Installer struct {
	Paths     *storage.Paths
	Config    *config.Config
	Client    *registry.Client
	Verbose   func(format string, args ...any)
	AgentTool string // agent type for --global install path (default: "claude")
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

	// 2. Build registry sources
	sources := make([]registry.RepoSource, len(inst.Config.Registries))
	for i, r := range inst.Config.Registries {
		sources[i] = registry.RepoSource{Name: r.Name, URL: r.URL, Token: r.Token, Username: r.Username, Branch: r.Branch}
	}

	if len(sources) == 0 {
		return fmt.Errorf("no registries configured; use 'skillhub repo add' to add one")
	}

	// 3. Fetch and merge all indexes
	inst.logVerbose("fetching indexes from %d registry(ies)", len(sources))
	idx, err := inst.Client.FetchAllIndexes(sources)
	if err != nil {
		return fmt.Errorf("fetching indexes: %w", err)
	}
	inst.logVerbose("found %d skill(s) across registries", len(idx.Skills))

	// 4. Find skill
	entry := idx.Find(name)
	if entry == nil {
		return fmt.Errorf("skill %q not found in any registry", name)
	}

	// 5. Resolve download URL and credentials
	var downloadURL string
	var token, username string
	for _, src := range sources {
		if src.Name == entry.Registry {
			downloadURL = src.ResolveDownloadURL(entry.DownloadURL)
			token = src.Token
			username = src.Username
			break
		}
	}
	if downloadURL == "" {
		downloadURL = entry.DownloadURL
	}

	// 6. Download archive
	cacheFile := filepath.Join(inst.Paths.CacheDir, fmt.Sprintf("%s-%s.tar.gz", name, entry.Version))
	inst.logVerbose("downloading %s to %s", downloadURL, cacheFile)
	if err := inst.Client.Download(downloadURL, cacheFile, token, username); err != nil {
		return fmt.Errorf("downloading skill: %w", err)
	}

	// 7. Verify checksum
	if entry.Checksum != "" {
		if err := VerifyChecksum(cacheFile, entry.Checksum); err != nil {
			os.Remove(cacheFile)
			return fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	// 8. Extract to temp directory
	tmpDir, err := os.MkdirTemp(inst.Paths.TmpDir, name+"-*")
	if err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	inst.logVerbose("extracting archive to %s", tmpDir)
	if err := ExtractTarGz(cacheFile, tmpDir); err != nil {
		return fmt.Errorf("extracting archive: %w", err)
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

	// 9.5. SKILL.md is required only for Claude Code integration (--global)
	if global {
		skillMDPath := filepath.Join(filepath.Dir(manifestPath), "SKILL.md")
		if _, err := os.Stat(skillMDPath); os.IsNotExist(err) {
			return fmt.Errorf("skill archive does not contain SKILL.md (required for Claude Code skill)")
		}
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
		projectRoot := storage.DetectProjectRoot()
		finalDir = filepath.Join(projectRoot, agent.SkillsPath, name)
	} else {
		finalDir = inst.Paths.SkillDir(name)
	}
	inst.logVerbose("installing to %s", finalDir)
	if err := os.MkdirAll(filepath.Dir(finalDir), 0755); err != nil {
		return fmt.Errorf("creating install directory: %w", err)
	}

	if force {
		os.RemoveAll(finalDir)
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
