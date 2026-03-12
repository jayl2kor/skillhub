package installer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeSHA256(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	if err := os.WriteFile(path, []byte("hello world\n"), 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := ComputeSHA256(path)
	if err != nil {
		t.Fatalf("ComputeSHA256: %v", err)
	}

	// sha256 of "hello world\n"
	expected := "a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447"
	if hash != expected {
		t.Errorf("expected %s, got %s", expected, hash)
	}
}

func TestVerifyChecksumMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	if err := os.WriteFile(path, []byte("hello world\n"), 0644); err != nil {
		t.Fatal(err)
	}

	err := VerifyChecksum(path, "sha256:a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestVerifyChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	if err := os.WriteFile(path, []byte("hello world\n"), 0644); err != nil {
		t.Fatal(err)
	}

	err := VerifyChecksum(path, "sha256:0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Error("expected error for checksum mismatch")
	}
}

func TestVerifyChecksumEmpty(t *testing.T) {
	// Empty checksum should pass (no verification)
	err := VerifyChecksum("/nonexistent", "")
	if err != nil {
		t.Errorf("expected no error for empty checksum, got: %v", err)
	}
}
