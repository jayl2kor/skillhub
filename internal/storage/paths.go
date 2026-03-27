// Package storage manages local skill directories and path resolution.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// Paths holds resolved filesystem paths for skillhub's data directories.
type Paths struct {
	Home        string
	Config      string
	SkillsDir   string
	CacheDir    string
	LogDir      string
	TmpDir      string
	ProjectRoot string // project root for local skill lookup; empty means auto-detect
}

// NewPaths constructs a Paths with all directories rooted under home.
func NewPaths(home string) *Paths {
	return &Paths{
		Home:      home,
		Config:    filepath.Join(home, "config.yaml"),
		SkillsDir: filepath.Join(home, "skills"),
		CacheDir:  filepath.Join(home, "cache"),
		LogDir:    filepath.Join(home, "logs"),
		TmpDir:    filepath.Join(home, "tmp"),
	}
}

// DefaultHome returns the skillhub home directory, using SKILLHUB_HOME if set,
// otherwise ~/.skillhub.
func DefaultHome() string {
	if env := os.Getenv("SKILLHUB_HOME"); env != "" {
		return env
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".skillhub")
	}

	return filepath.Join(home, ".skillhub")
}

// EnsureDirectories creates all required skillhub directories if they do not exist.
func (p *Paths) EnsureDirectories() error {
	dirs := []string{p.Home, p.SkillsDir, p.CacheDir, p.LogDir, p.TmpDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}
	return nil
}

// SkillDir returns the installation directory for the skill with the given name.
func (p *Paths) SkillDir(name string) string {
	return filepath.Join(p.SkillsDir, name)
}
