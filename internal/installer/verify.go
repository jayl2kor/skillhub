package installer

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"
)

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
