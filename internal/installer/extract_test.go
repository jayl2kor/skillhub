package installer

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func createTestArchive(t *testing.T, files map[string]string) string {
	t.Helper()

	archivePath := filepath.Join(t.TempDir(), "test.tar.gz")
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}

	return archivePath
}

func createTestArchiveWithSymlink(t *testing.T) string {
	t.Helper()

	archivePath := filepath.Join(t.TempDir(), "test.tar.gz")
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	header := &tar.Header{
		Name:     "link",
		Typeflag: tar.TypeSymlink,
		Linkname: "/etc/passwd",
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatal(err)
	}

	return archivePath
}

func TestExtractTarGz(t *testing.T) {
	archive := createTestArchive(t, map[string]string{
		"skill.json": `{"name": "test"}`,
		"prompt.md":  "# Test",
	})

	dest := t.TempDir()
	if err := ExtractTarGz(archive, dest); err != nil {
		t.Fatalf("ExtractTarGz: %v", err)
	}

	// Check files exist
	data, err := os.ReadFile(filepath.Join(dest, "skill.json"))
	if err != nil {
		t.Fatalf("reading skill.json: %v", err)
	}
	if string(data) != `{"name": "test"}` {
		t.Errorf("unexpected content: %q", string(data))
	}
}

func TestExtractTarGzPathTraversal(t *testing.T) {
	archive := createTestArchive(t, map[string]string{
		"../escape.txt": "malicious",
	})

	dest := t.TempDir()
	err := ExtractTarGz(archive, dest)
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestExtractTarGzAbsolutePath(t *testing.T) {
	archive := createTestArchive(t, map[string]string{
		"/etc/passwd": "malicious",
	})

	dest := t.TempDir()
	err := ExtractTarGz(archive, dest)
	if err == nil {
		t.Error("expected error for absolute path")
	}
}

func TestExtractTarGzSymlink(t *testing.T) {
	archive := createTestArchiveWithSymlink(t)

	dest := t.TempDir()
	err := ExtractTarGz(archive, dest)
	if err == nil {
		t.Error("expected error for symlink")
	}
}

func TestExtractTarGzSubdirectory(t *testing.T) {
	archive := createTestArchive(t, map[string]string{
		"sub/file.txt": "content",
	})

	dest := t.TempDir()
	if err := ExtractTarGz(archive, dest); err != nil {
		t.Fatalf("ExtractTarGz: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dest, "sub", "file.txt"))
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	if string(data) != "content" {
		t.Errorf("unexpected content: %q", string(data))
	}
}
