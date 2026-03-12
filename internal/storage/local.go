package storage

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jayl2kor/skillhub/internal/skill"
)

// DetectProjectRoot returns the git repository root, or the current working directory if not in a git repo.
func DetectProjectRoot() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	cwd, _ := os.Getwd()
	return cwd
}

func (p *Paths) projectRoot() string {
	if p.ProjectRoot != "" {
		return p.ProjectRoot
	}
	return DetectProjectRoot()
}

func ListInstalledSkills(paths *Paths) ([]skill.InstalledSkill, error) {
	seen := make(map[string]bool)
	var skills []skill.InstalledSkill

	// Global skills
	if entries, err := os.ReadDir(paths.SkillsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillDir := filepath.Join(paths.SkillsDir, entry.Name())
			if s, err := loadInstalledSkill(skillDir); err == nil {
				seen[s.Manifest.Name] = true
				skills = append(skills, *s)
			}
		}
	}

	// Local project skills (all agent paths)
	projectRoot := paths.projectRoot()
	for _, agentPath := range skill.AllAgentSkillsPaths() {
		localDir := filepath.Join(projectRoot, agentPath)
		entries, err := os.ReadDir(localDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			if seen[entry.Name()] {
				continue
			}
			skillDir := filepath.Join(localDir, entry.Name())
			if s, err := loadInstalledSkill(skillDir); err == nil {
				seen[s.Manifest.Name] = true
				skills = append(skills, *s)
			}
		}
	}

	return skills, nil
}

func IsInstalled(paths *Paths, name string) bool {
	// Global check
	manifestPath := filepath.Join(paths.SkillsDir, name, "skill.json")
	if _, err := os.Stat(manifestPath); err == nil {
		return true
	}
	// Local check (all agent skill paths)
	projectRoot := paths.projectRoot()
	for _, agentPath := range skill.AllAgentSkillsPaths() {
		localManifest := filepath.Join(projectRoot, agentPath, name, "skill.json")
		if _, err := os.Stat(localManifest); err == nil {
			return true
		}
	}
	return false
}

func GetInstalledSkill(paths *Paths, name string) (*skill.InstalledSkill, error) {
	// Check global path first
	skillDir := filepath.Join(paths.SkillsDir, name)
	manifestPath := filepath.Join(skillDir, "skill.json")

	if _, err := os.Stat(manifestPath); err == nil {
		return loadInstalledSkill(skillDir)
	}

	// Check local project agent paths
	projectRoot := paths.projectRoot()
	for _, agentPath := range skill.AllAgentSkillsPaths() {
		localDir := filepath.Join(projectRoot, agentPath, name)
		localManifest := filepath.Join(localDir, "skill.json")
		if _, err := os.Stat(localManifest); err == nil {
			return loadInstalledSkill(localDir)
		}
	}

	return nil, fmt.Errorf("skill %q is not installed", name)
}

func loadInstalledSkill(skillDir string) (*skill.InstalledSkill, error) {
	manifestPath := filepath.Join(skillDir, "skill.json")
	m, err := skill.LoadManifest(manifestPath)
	if err != nil {
		return nil, err
	}

	installed := &skill.InstalledSkill{
		Manifest: *m,
		Dir:      skillDir,
	}

	metaPath := filepath.Join(skillDir, ".install.json")
	if meta, err := skill.LoadInstallMeta(metaPath); err == nil {
		installed.Meta = *meta
	}

	return installed, nil
}
