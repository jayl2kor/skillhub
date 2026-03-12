package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateValid(t *testing.T) {
	m := &Manifest{
		Name:        "code-review",
		Version:     "1.0.0",
		Description: "Review code changes",
		Entry:       "prompt.md",
		Type:        "prompt",
	}

	if err := m.Validate(); err != nil {
		t.Errorf("expected valid manifest, got error: %v", err)
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"code-review", false},
		{"my-skill", false},
		{"a", false},
		{"a1", false},
		{"", true},
		{"Code-Review", true},
		{"code_review", true},
		{"code review", true},
		{"-invalid", true},
		{"invalid-", true},
		{"a--b", true},
	}

	for _, tt := range tests {
		m := &Manifest{
			Name:        tt.name,
			Version:     "1.0.0",
			Description: "desc",
			Entry:       "prompt.md",
			Type:        "prompt",
		}
		err := m.Validate()
		if (err != nil) != tt.wantErr {
			t.Errorf("name=%q: wantErr=%v, got %v", tt.name, tt.wantErr, err)
		}
	}
}

func TestValidateVersion(t *testing.T) {
	tests := []struct {
		version string
		wantErr bool
	}{
		{"1.0.0", false},
		{"0.9.1", false},
		{"10.20.30", false},
		{"", true},
		{"1.0", true},
		{"1.0.0.0", true},
		{"v1.0.0", true},
		{"abc", true},
	}

	for _, tt := range tests {
		m := &Manifest{
			Name:        "test",
			Version:     tt.version,
			Description: "desc",
			Entry:       "prompt.md",
			Type:        "prompt",
		}
		err := m.Validate()
		if (err != nil) != tt.wantErr {
			t.Errorf("version=%q: wantErr=%v, got %v", tt.version, tt.wantErr, err)
		}
	}
}

func TestValidateEntry(t *testing.T) {
	tests := []struct {
		entry   string
		wantErr bool
	}{
		{"prompt.md", false},
		{"sub/prompt.md", false},
		{"", true},
		{"/absolute/path", true},
		{"../escape", true},
		{"sub/../../escape", true},
	}

	for _, tt := range tests {
		m := &Manifest{
			Name:        "test",
			Version:     "1.0.0",
			Description: "desc",
			Entry:       tt.entry,
			Type:        "prompt",
		}
		err := m.Validate()
		if (err != nil) != tt.wantErr {
			t.Errorf("entry=%q: wantErr=%v, got %v", tt.entry, tt.wantErr, err)
		}
	}
}

func TestValidateType(t *testing.T) {
	tests := []struct {
		typ     string
		wantErr bool
	}{
		{"prompt", false},
		{"shell", false},
		{"python", false},
		{"node", false},
		{"", true},
		{"unknown", true},
	}

	for _, tt := range tests {
		m := &Manifest{
			Name:        "test",
			Version:     "1.0.0",
			Description: "desc",
			Entry:       "prompt.md",
			Type:        tt.typ,
		}
		err := m.Validate()
		if (err != nil) != tt.wantErr {
			t.Errorf("type=%q: wantErr=%v, got %v", tt.typ, tt.wantErr, err)
		}
	}
}

func TestLoadManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skill.json")

	content := `{
		"name": "test-skill",
		"version": "1.0.0",
		"description": "A test skill",
		"entry": "prompt.md",
		"type": "prompt",
		"tags": ["test"]
	}`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.Name != "test-skill" {
		t.Errorf("expected name 'test-skill', got %q", m.Name)
	}
	if m.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", m.Version)
	}
	if len(m.Tags) != 1 || m.Tags[0] != "test" {
		t.Errorf("expected tags [test], got %v", m.Tags)
	}
}

func TestLoadManifestInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "skill.json")

	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadManifest(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadManifestMissing(t *testing.T) {
	_, err := LoadManifest("/nonexistent/skill.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
