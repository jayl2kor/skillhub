// Package storage manages local skill directories and path resolution.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// Paths holds the directory layout for skillhub's storage.
type Paths struct {
	Home        string
	Config      string
	SkillsDir   string
	CacheDir    string
	LogDir      string
	TmpDir      string
	ProjectRoot string // project root for local skill lookup; empty means auto-detect
}

// NewPaths returns a Paths rooted at the given home directory.
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

// DefaultHome returns the default skillhub home directory path.
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

// EnsureDirectories creates all storage directories if they do not exist.
func (p *Paths) EnsureDirectories() error {
	dirs := []string{p.Home, p.SkillsDir, p.CacheDir, p.LogDir, p.TmpDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}
	return nil
}

// SkillDir returns the directory path for the named skill.
func (p *Paths) SkillDir(name string) string {
	return filepath.Join(p.SkillsDir, name)
}
