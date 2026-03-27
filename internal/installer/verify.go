package installer

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"
)

// VerifyChecksum checks that the SHA-256 of the file matches the expected value.
func VerifyChecksum(filePath string, expected string) error {
	if expected == "" {
		return nil
	}

	expected = strings.TrimPrefix(expected, "sha256:")

	actual, err := ComputeSHA256(filePath)
	if err != nil {
		return err
	}

	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}

	return nil
}

// ComputeSHA256 returns the hex-encoded SHA-256 digest of the file.
func ComputeSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("opening file for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("computing checksum: %w", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
