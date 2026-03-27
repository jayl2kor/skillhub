package skill

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// InstalledSkill represents a skill that has been installed locally.
type InstalledSkill struct {
	Manifest Manifest
	Dir      string
	Meta     InstallMeta
}

// InstallMeta holds metadata about how and when a skill was installed.
type InstallMeta struct {
	InstalledAt string `json:"installed_at"`
	Registry    string `json:"registry"`
	Version     string `json:"version"`
	Checksum    string `json:"checksum,omitempty"`
}

// NewInstallMeta creates an InstallMeta with the current timestamp.
func NewInstallMeta(registry, version, checksum string) InstallMeta {
	return InstallMeta{
		InstalledAt: time.Now().UTC().Format(time.RFC3339),
		Registry:    registry,
		Version:     version,
		Checksum:    checksum,
	}
}

// LoadInstallMeta reads and parses install metadata from the given file path.
func LoadInstallMeta(path string) (*InstallMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading install meta: %w", err)
	}

	var meta InstallMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parsing install meta: %w", err)
	}

	return &meta, nil
}

// Save writes the install metadata to the given file path as JSON.
func (m *InstallMeta) Save(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling install meta: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing install meta: %w", err)
	}

	return nil
}
