package registry

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchIndexLocal(t *testing.T) {
	dir := t.TempDir()

	indexData := `{"skills": [{"name": "test", "version": "1.0.0", "description": "test skill", "download_url": "test.tar.gz"}]}`
	if err := os.WriteFile(filepath.Join(dir, "index.json"), []byte(indexData), 0644); err != nil {
		t.Fatal(err)
	}

	client := NewClient()
	source := &RepoSource{Name: "local", URL: dir}

	idx, err := client.FetchIndex(source)
	if err != nil {
		t.Fatalf("FetchIndex: %v", err)
	}

	if len(idx.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(idx.Skills))
	}

	if idx.Skills[0].Name != "test" {
		t.Errorf("expected name 'test', got %q", idx.Skills[0].Name)
	}

	if idx.Skills[0].Registry != "local" {
		t.Errorf("expected registry 'local', got %q", idx.Skills[0].Registry)
	}
}

func TestFetchIndexHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"skills": [{"name": "remote", "version": "2.0.0", "description": "remote skill", "download_url": "remote.tar.gz"}]}`))
	}))
	defer server.Close()

	client := NewClient()

	// Direct fetch test
	data, err := client.fetch(server.URL, "", "")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}

	parsedIdx, err := ParseIndex(data)
	if err != nil {
		t.Fatalf("ParseIndex: %v", err)
	}

	if len(parsedIdx.Skills) != 1 || parsedIdx.Skills[0].Name != "remote" {
		t.Errorf("unexpected index content")
	}
}

func TestFetchAllIndexes(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	os.WriteFile(filepath.Join(dir1, "index.json"),
		[]byte(`{"skills": [{"name": "a", "version": "1.0.0", "description": "skill a", "download_url": "a.tar.gz"}]}`), 0644)
	os.WriteFile(filepath.Join(dir2, "index.json"),
		[]byte(`{"skills": [{"name": "b", "version": "2.0.0", "description": "skill b", "download_url": "b.tar.gz"}]}`), 0644)

	client := NewClient()
	sources := []RepoSource{
		{Name: "reg1", URL: dir1},
		{Name: "reg2", URL: dir2},
	}

	idx, err := client.FetchAllIndexes(sources)
	if err != nil {
		t.Fatalf("FetchAllIndexes: %v", err)
	}

	if len(idx.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(idx.Skills))
	}
}

func TestDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("file content"))
	}))
	defer server.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "downloaded")

	client := NewClient()
	if err := client.Download(server.URL, dest, "", ""); err != nil {
		t.Fatalf("Download: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading downloaded file: %v", err)
	}

	if string(data) != "file content" {
		t.Errorf("expected 'file content', got %q", string(data))
	}
}
