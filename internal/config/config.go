// Package config manages skillhub configuration loading and persistence.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// RegistryEntry holds the configuration for a single remote skill registry.
type RegistryEntry struct {
	Name         string `yaml:"name"`
	URL          string `yaml:"url"`
	Token        string `yaml:"token,omitempty"`
	Username     string `yaml:"username,omitempty"`
	Branch       string `yaml:"branch,omitempty"`
	SkillsPrefix string `yaml:"skills_prefix,omitempty"`
}

// Config holds the user's skillhub configuration.
type Config struct {
	Registries []RegistryEntry `yaml:"registries"`
	InstallDir string          `yaml:"install_dir"`
	CacheDir   string          `yaml:"cache_dir"`
	LogDir     string          `yaml:"log_dir"`
}

// DefaultConfig returns a Config with default directory paths rooted at home.
func DefaultConfig(home string) *Config {
	return &Config{
		Registries: []RegistryEntry{},
		InstallDir: filepath.Join(home, "skills"),
		CacheDir:   filepath.Join(home, "cache"),
		LogDir:     filepath.Join(home, "logs"),
	}
}

// Load reads and parses the YAML configuration file at the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

// Save writes the configuration to the given path as YAML.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// Validate checks that all registry entries have required fields.
func (c *Config) Validate() error {
	for i, r := range c.Registries {
		if r.Name == "" {
			return fmt.Errorf("registry[%d]: name is required", i)
		}
		if r.URL == "" {
			return fmt.Errorf("registry[%d] %q: url is required", i, r.Name)
		}
	}
	return nil
}

// AddRegistry upserts a registry entry by name, updating it if it already exists.
func (c *Config) AddRegistry(name, rawURL, token, username, branch, skillsPrefix string) error {
	for i, r := range c.Registries {
		if r.Name == name {
			c.Registries[i].URL = rawURL
			c.Registries[i].Token = token
			c.Registries[i].Username = username
			c.Registries[i].Branch = branch
			c.Registries[i].SkillsPrefix = skillsPrefix
			return nil
		}
	}
	c.Registries = append(c.Registries, RegistryEntry{Name: name, URL: rawURL, Token: token, Username: username, Branch: branch, SkillsPrefix: skillsPrefix})
	return nil
}

// RemoveRegistry removes the registry with the given name, returning an error if not found.
func (c *Config) RemoveRegistry(name string) error {
	for i, r := range c.Registries {
		if r.Name == name {
			c.Registries = append(c.Registries[:i], c.Registries[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("registry %q not found", name)
}
