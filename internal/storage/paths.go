// Package storage manages local skill directories and path resolution.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

type Paths struct {
	Home        string
	Config      string
	SkillsDir   string
	CacheDir    string
	LogDir      string
	TmpDir      string
	ProjectRoot string // project root for local skill lookup; empty means auto-detect
}

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

func (p *Paths) EnsureDirectories() error {
	dirs := []string{p.Home, p.SkillsDir, p.CacheDir, p.LogDir, p.TmpDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}
	return nil
}

func (p *Paths) SkillDir(name string) string {
	return filepath.Join(p.SkillsDir, name)
}
