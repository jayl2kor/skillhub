// Package skill defines skill manifests, versioning, and installation metadata.
package skill

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	nameRegex    = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	versionRegex = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	validTypes   = map[string]bool{
		"prompt": true,
		"shell":  true,
		"python": true,
		"node":   true,
	}
)

// Manifest describes a skill's metadata as declared in skill.json.
type Manifest struct {
	Name             string   `json:"name"`
	Version          string   `json:"version"`
	Description      string   `json:"description"`
	Entry            string   `json:"entry"`
	Type             string   `json:"type"`
	Tags             []string `json:"tags,omitempty"`
	Author           string   `json:"author,omitempty"`
	Homepage         string   `json:"homepage,omitempty"`
	License          string   `json:"license,omitempty"`
	CompatibleAgents []string `json:"compatible_agents,omitempty"`
}

// LoadManifest reads and parses a skill manifest from the given file path.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	return &m, nil
}

// Validate checks that all required manifest fields are present and well-formed.
func (m *Manifest) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("manifest: name is required")
	}
	if !nameRegex.MatchString(m.Name) {
		return fmt.Errorf("manifest: name must be lowercase alphanumeric with hyphens, got %q", m.Name)
	}

	if m.Version == "" {
		return fmt.Errorf("manifest: version is required")
	}
	if !versionRegex.MatchString(m.Version) {
		return fmt.Errorf("manifest: version must be semver (X.Y.Z), got %q", m.Version)
	}

	if m.Description == "" {
		return fmt.Errorf("manifest: description is required")
	}

	if m.Entry == "" {
		return fmt.Errorf("manifest: entry is required")
	}
	if filepath.IsAbs(m.Entry) {
		return fmt.Errorf("manifest: entry must be a relative path, got %q", m.Entry)
	}
	cleaned := filepath.Clean(m.Entry)
	if strings.HasPrefix(cleaned, "..") {
		return fmt.Errorf("manifest: entry must not escape skill directory, got %q", m.Entry)
	}

	if m.Type == "" {
		return fmt.Errorf("manifest: type is required")
	}
	if !validTypes[m.Type] {
		return fmt.Errorf("manifest: unsupported type %q", m.Type)
	}

	return nil
}
