package registry

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestContentsAPIPutURL(t *testing.T) {
	tests := []struct {
		name     string
		source   RepoSource
		path     string
		expected string
	}{
		{
			name:     "github.com",
			source:   RepoSource{URL: "https://github.com/owner/repo"},
			path:     "my-skill-1.0.0.tar.gz",
			expected: "https://api.github.com/repos/owner/repo/contents/my-skill-1.0.0.tar.gz",
		},
		{
			name:     "github enterprise",
			source:   RepoSource{URL: "https://git.corp.com/owner/repo"},
			path:     "index.json",
			expected: "https://git.corp.com/api/v3/repos/owner/repo/contents/index.json",
		},
		{
			name:     "local path",
			source:   RepoSource{URL: "/tmp/registry"},
			path:     "my-skill-1.0.0.tar.gz",
			expected: "/tmp/registry/my-skill-1.0.0.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.source.ContentsAPIPutURL(tt.path)
			if got != tt.expected {
				t.Errorf("ContentsAPIPutURL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIsLocal(t *testing.T) {
	tests := []struct {
		url    string
		expect bool
	}{
		{"/tmp/registry", true},
		{"./local", true},
		{"../parent", true},
		{"https://github.com/owner/repo", false},
	}
	for _, tt := range tests {
		s := &RepoSource{URL: tt.url}
		if got := s.IsLocal(); got != tt.expect {
			t.Errorf("IsLocal(%q) = %v, want %v", tt.url, got, tt.expect)
		}
	}
}

func TestGetFileSHA(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v3/repos/owner/repo/contents/exists.tar.gz":
			w.Header().Set("Content-Type", "application/json")
			resp, _ := json.Marshal(githubFileInfo{SHA: "abc123"})
			if _, err := w.Write(resp); err != nil {
				t.Errorf("writing response: %v", err)
			}
		case "/api/v3/repos/owner/repo/contents/missing.tar.gz":
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient()
	source := &RepoSource{
		URL:    server.URL + "/owner/repo",
		Branch: "main",
	}

	// Existing file returns SHA
	sha, err := client.GetFileSHA(source, "exists.tar.gz")
	if err != nil {
		t.Fatalf("GetFileSHA existing: %v", err)
	}
	if sha != "abc123" {
		t.Errorf("expected sha abc123, got %q", sha)
	}

	// Missing file returns empty string
	sha, err = client.GetFileSHA(source, "missing.tar.gz")
	if err != nil {
		t.Fatalf("GetFileSHA missing: %v", err)
	}
	if sha != "" {
		t.Errorf("expected empty sha for missing file, got %q", sha)
	}
}

func TestUploadFileGitHub(t *testing.T) {
	var receivedBody githubPutRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("reading body: %v", err)
		}
		if err := json.Unmarshal(body, &receivedBody); err != nil {
			t.Fatalf("parsing body: %v", err)
		}

		w.WriteHeader(http.StatusCreated)
		if _, err := w.Write([]byte(`{"content":{"sha":"new123"}}`)); err != nil {
			t.Errorf("writing response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient()
	source := &RepoSource{
		URL:    server.URL + "/owner/repo",
		Branch: "main",
		Token:  "test-token",
	}

	content := []byte("archive content")
	err := client.UploadFile(source, "skill-1.0.0.tar.gz", content, "", "publish skill@1.0.0")
	if err != nil {
		t.Fatalf("UploadFile: %v", err)
	}

	// Verify request body
	decoded, err := base64.StdEncoding.DecodeString(receivedBody.Content)
	if err != nil {
		t.Fatalf("decoding content: %v", err)
	}
	if string(decoded) != "archive content" {
		t.Errorf("content = %q, want %q", string(decoded), "archive content")
	}
	if receivedBody.Branch != "main" {
		t.Errorf("branch = %q, want %q", receivedBody.Branch, "main")
	}
	if receivedBody.Message != "publish skill@1.0.0" {
		t.Errorf("message = %q, want %q", receivedBody.Message, "publish skill@1.0.0")
	}
}

func TestUploadFileLocal(t *testing.T) {
	dir := t.TempDir()
	client := NewClient()
	source := &RepoSource{URL: dir}

	content := []byte("local archive content")
	err := client.UploadFile(source, "skill-1.0.0.tar.gz", content, "", "")
	if err != nil {
		t.Fatalf("UploadFile local: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "skill-1.0.0.tar.gz"))
	if err != nil {
		t.Fatalf("reading uploaded file: %v", err)
	}
	if string(data) != "local archive content" {
		t.Errorf("content = %q, want %q", string(data), "local archive content")
	}
}

func TestUpdateIndexLocal(t *testing.T) {
	dir := t.TempDir()
	client := NewClient()
	source := &RepoSource{Name: "test", URL: dir}

	entry := IndexEntry{
		Name:        "my-skill",
		Version:     "1.0.0",
		Description: "test skill",
		DownloadURL: "my-skill-1.0.0.tar.gz",
		Checksum:    "sha256:abc",
	}

	// First publish: creates index.json
	if err := client.UpdateIndex(source, entry, false); err != nil {
		t.Fatalf("UpdateIndex create: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		t.Fatalf("reading index: %v", err)
	}
	idx, err := ParseIndex(data)
	if err != nil {
		t.Fatalf("parsing index: %v", err)
	}
	if len(idx.Skills) != 1 || idx.Skills[0].Name != "my-skill" {
		t.Errorf("unexpected index content: %+v", idx.Skills)
	}

	// Duplicate without force: should error
	err = client.UpdateIndex(source, entry, false)
	if err == nil {
		t.Fatal("expected error for duplicate version without force")
	}

	// Duplicate with force: should succeed
	entry.Description = "updated skill"
	if err := client.UpdateIndex(source, entry, true); err != nil {
		t.Fatalf("UpdateIndex force: %v", err)
	}

	data, err = os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		t.Fatalf("reading index: %v", err)
	}
	idx, err = ParseIndex(data)
	if err != nil {
		t.Fatalf("parsing index: %v", err)
	}
	if len(idx.Skills) != 1 {
		t.Errorf("expected 1 skill after force update, got %d", len(idx.Skills))
	}
	if idx.Skills[0].Description != "updated skill" {
		t.Errorf("description = %q, want %q", idx.Skills[0].Description, "updated skill")
	}

	// Add different version: should append
	entry2 := IndexEntry{
		Name:        "my-skill",
		Version:     "2.0.0",
		Description: "v2",
		DownloadURL: "my-skill-2.0.0.tar.gz",
	}
	if err := client.UpdateIndex(source, entry2, false); err != nil {
		t.Fatalf("UpdateIndex new version: %v", err)
	}

	data, err = os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		t.Fatalf("reading index: %v", err)
	}
	idx, err = ParseIndex(data)
	if err != nil {
		t.Fatalf("parsing index: %v", err)
	}
	if len(idx.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(idx.Skills))
	}
}
