package registry

import (
	"encoding/json"
	"fmt"
	"strings"
)

// IndexEntry describes a single skill in a registry index.
type IndexEntry struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Tags        []string `json:"tags,omitempty"`
	DownloadURL string   `json:"download_url"`
	Checksum    string   `json:"checksum,omitempty"`
	Registry    string   `json:"-"`
}

// Index holds the full list of skills available in a registry.
type Index struct {
	Skills []IndexEntry `json:"skills"`
}

// ParseIndex unmarshals JSON data into an Index.
func ParseIndex(data []byte) (*Index, error) {
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parsing index: %w", err)
	}
	return &idx, nil
}

// Search returns entries whose name, description, or tags match the query.
func (idx *Index) Search(query string) []IndexEntry {
	if query == "*" || query == "" {
		return idx.Skills
	}

	query = strings.ToLower(query)
	var results []IndexEntry

	for _, entry := range idx.Skills {
		if matchesQuery(entry, query) {
			results = append(results, entry)
		}
	}

	return results
}

func matchesQuery(entry IndexEntry, query string) bool {
	if strings.Contains(strings.ToLower(entry.Name), query) {
		return true
	}
	if strings.Contains(strings.ToLower(entry.Description), query) {
		return true
	}
	for _, tag := range entry.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	return false
}

// Find returns the first entry with the given name, or nil if not found.
func (idx *Index) Find(name string) *IndexEntry {
	for _, entry := range idx.Skills {
		if entry.Name == name {
			return &entry
		}
	}
	return nil
}

// FindVersion returns an entry matching both name and version.
func (idx *Index) FindVersion(name, version string) *IndexEntry {
	for _, entry := range idx.Skills {
		if entry.Name == name && entry.Version == version {
			return &entry
		}
	}
	return nil
}

// MergeIndexes combines multiple indexes into a single unified index.
func MergeIndexes(indexes ...*Index) *Index {
	merged := &Index{}
	for _, idx := range indexes {
		if idx != nil {
			merged.Skills = append(merged.Skills, idx.Skills...)
		}
	}
	return merged
}
