package storage

import (
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

func ListInstalledSkills(paths *Paths) ([]skill.InstalledSkill, error) {
	entries, err := os.ReadDir(paths.SkillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var skills []skill.InstalledSkill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(paths.SkillsDir, entry.Name())
		manifestPath := filepath.Join(skillDir, "skill.json")

		m, err := skill.LoadManifest(manifestPath)
		if err != nil {
			continue
		}

		installed := skill.InstalledSkill{
			Manifest: *m,
			Dir:      skillDir,
		}

		metaPath := filepath.Join(skillDir, ".install.json")
		if meta, err := skill.LoadInstallMeta(metaPath); err == nil {
			installed.Meta = *meta
		}

		skills = append(skills, installed)
	}

	return skills, nil
}

func IsInstalled(paths *Paths, name string) bool {
	// Global check (existing)
	manifestPath := filepath.Join(paths.SkillsDir, name, "skill.json")
	if _, err := os.Stat(manifestPath); err == nil {
		return true
	}
	// Local check (.claude/skills/)
	projectRoot := DetectProjectRoot()
	skillMD := filepath.Join(projectRoot, ".claude", "skills", name, "SKILL.md")
	_, err := os.Stat(skillMD)
	return err == nil
}

func GetInstalledSkill(paths *Paths, name string) (*skill.InstalledSkill, error) {
	skillDir := filepath.Join(paths.SkillsDir, name)
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
