package registry

import (
	"context"
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
			path:     "skills/my-skill/skill.json",
			expected: "https://api.github.com/repos/owner/repo/contents/skills/my-skill/skill.json",
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
			path:     "skills/my-skill/skill.json",
			expected: "/tmp/registry/skills/my-skill/skill.json",
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
		case "/api/v3/repos/owner/repo/contents/skills/my-skill/skill.json":
			w.Header().Set("Content-Type", "application/json")
			resp, _ := json.Marshal(githubFileInfo{SHA: "abc123"})
			if _, err := w.Write(resp); err != nil {
				t.Errorf("writing response: %v", err)
			}
		case "/api/v3/repos/owner/repo/contents/skills/my-skill/missing.md":
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

	ctx := context.Background()
	sha, err := client.GetFileSHA(ctx, source, "skills/my-skill/skill.json")
	if err != nil {
		t.Fatalf("GetFileSHA existing: %v", err)
	}
	if sha != "abc123" {
		t.Errorf("expected sha abc123, got %q", sha)
	}

	sha, err = client.GetFileSHA(ctx, source, "skills/my-skill/missing.md")
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

	ctx := context.Background()
	content := []byte(`{"name":"my-skill"}`)
	err := client.UploadFile(ctx, source, "skills/my-skill/skill.json", content, "", "publish my-skill@1.0.0")
	if err != nil {
		t.Fatalf("UploadFile: %v", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(receivedBody.Content)
	if err != nil {
		t.Fatalf("decoding content: %v", err)
	}
	if string(decoded) != `{"name":"my-skill"}` {
		t.Errorf("content = %q, want %q", string(decoded), `{"name":"my-skill"}`)
	}
	if receivedBody.Branch != "main" {
		t.Errorf("branch = %q, want %q", receivedBody.Branch, "main")
	}
	if receivedBody.Message != "publish my-skill@1.0.0" {
		t.Errorf("message = %q, want %q", receivedBody.Message, "publish my-skill@1.0.0")
	}
}

func TestUploadFileLocal(t *testing.T) {
	dir := t.TempDir()
	client := NewClient()
	source := &RepoSource{URL: dir}

	ctx := context.Background()
	content := []byte(`{"name":"my-skill"}`)
	err := client.UploadFile(ctx, source, "skills/my-skill/skill.json", content, "", "")
	if err != nil {
		t.Fatalf("UploadFile local: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "skills", "my-skill", "skill.json"))
	if err != nil {
		t.Fatalf("reading uploaded file: %v", err)
	}
	if string(data) != `{"name":"my-skill"}` {
		t.Errorf("content = %q, want %q", string(data), `{"name":"my-skill"}`)
	}
}

func TestUploadDirectoryLocal(t *testing.T) {
	// Create source skill directory
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "skill.json"), []byte(`{"name":"test"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("# Test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(srcDir, "refs"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "refs", "guide.md"), []byte("guide"), 0644); err != nil {
		t.Fatal(err)
	}
	// Hidden file should be skipped
	if err := os.WriteFile(filepath.Join(srcDir, ".DS_Store"), []byte("hidden"), 0644); err != nil {
		t.Fatal(err)
	}

	regDir := t.TempDir()
	client := NewClient()
	ctx := context.Background()
	source := &RepoSource{Name: "local", URL: regDir}

	if err := client.UploadDirectory(ctx, source, srcDir, "skills/test", "publish"); err != nil {
		t.Fatalf("UploadDirectory: %v", err)
	}

	// Verify files were copied
	data, err := os.ReadFile(filepath.Join(regDir, "skills", "test", "skill.json"))
	if err != nil {
		t.Fatalf("reading skill.json: %v", err)
	}
	if string(data) != `{"name":"test"}` {
		t.Errorf("skill.json content = %q", string(data))
	}

	data, err = os.ReadFile(filepath.Join(regDir, "skills", "test", "refs", "guide.md"))
	if err != nil {
		t.Fatalf("reading refs/guide.md: %v", err)
	}
	if string(data) != "guide" {
		t.Errorf("guide.md content = %q", string(data))
	}

	// Hidden file should not be copied
	if _, err := os.Stat(filepath.Join(regDir, "skills", "test", ".DS_Store")); !os.IsNotExist(err) {
		t.Error(".DS_Store should not be copied")
	}
}

func TestUploadDirectoryGitHub(t *testing.T) {
	uploaded := make(map[string]string) // remotePath -> content

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			var body githubPutRequest
			data, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(data, &body); err != nil {
				t.Errorf("parsing PUT body: %v", err)
			}
			decoded, _ := base64.StdEncoding.DecodeString(body.Content)
			uploaded[r.URL.Path] = string(decoded)

			w.WriteHeader(http.StatusCreated)
			if _, err := w.Write([]byte(`{"content":{"sha":"new"}}`)); err != nil {
				t.Errorf("writing response: %v", err)
			}
			return
		}
		// GET for SHA check returns 404 (new files)
		http.NotFound(w, r)
	}))
	defer server.Close()

	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "skill.json"), []byte(`{"name":"test"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("# Test"), 0644); err != nil {
		t.Fatal(err)
	}

	client := NewClient()
	source := &RepoSource{
		URL:    server.URL + "/owner/repo",
		Branch: "main",
		Token:  "tok",
	}

	ctx := context.Background()
	if err := client.UploadDirectory(ctx, source, srcDir, "skills/test", "publish test@1.0.0"); err != nil {
		t.Fatalf("UploadDirectory GitHub: %v", err)
	}

	if v, ok := uploaded["/api/v3/repos/owner/repo/contents/skills/test/skill.json"]; !ok {
		t.Error("skill.json not uploaded")
	} else if v != `{"name":"test"}` {
		t.Errorf("skill.json content = %q", v)
	}

	if v, ok := uploaded["/api/v3/repos/owner/repo/contents/skills/test/SKILL.md"]; !ok {
		t.Error("SKILL.md not uploaded")
	} else if v != "# Test" {
		t.Errorf("SKILL.md content = %q", v)
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
		DownloadURL: "skills/my-skill/",
	}

	ctx := context.Background()

	// First publish: creates index.json
	if err := client.UpdateIndex(ctx, source, entry, false); err != nil {
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
	if idx.Skills[0].DownloadURL != "skills/my-skill/" {
		t.Errorf("download_url = %q, want %q", idx.Skills[0].DownloadURL, "skills/my-skill/")
	}

	// Duplicate without force: should error
	err = client.UpdateIndex(ctx, source, entry, false)
	if err == nil {
		t.Fatal("expected error for duplicate version without force")
	}

	// Duplicate with force: should succeed
	entry.Description = "updated skill"
	if err := client.UpdateIndex(ctx, source, entry, true); err != nil {
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
		DownloadURL: "skills/my-skill/",
	}
	if err := client.UpdateIndex(ctx, source, entry2, false); err != nil {
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
