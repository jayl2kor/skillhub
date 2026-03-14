package installer

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxFileSize = 100 * 1024 * 1024 // 100MB

// ExtractTarGz extracts a gzip-compressed tar archive into destDir.
func ExtractTarGz(archivePath string, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar entry: %w", err)
		}

		name, err := sanitizePath(header.Name, destDir)
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("creating directory %s: %w", name, err)
			}

		case tar.TypeReg:
			if header.Size > maxFileSize {
				return fmt.Errorf("file %s exceeds maximum size (%d bytes)", name, maxFileSize)
			}

			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("creating parent directory for %s: %w", name, err)
			}

			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return fmt.Errorf("creating file %s: %w", name, err)
			}

			if _, err := io.Copy(outFile, io.LimitReader(tr, header.Size)); err != nil {
				outFile.Close()
				return fmt.Errorf("writing file %s: %w", name, err)
			}
			outFile.Close()

		case tar.TypeSymlink, tar.TypeLink:
			return fmt.Errorf("archive contains unsupported link: %s", name)

		default:
			// Skip unknown types
		}
	}

	return nil
}

func sanitizePath(name string, destDir string) (string, error) {
	// Clean the path
	cleaned := filepath.Clean(name)

	// Reject absolute paths (before stripping slash)
	if filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("archive contains absolute path: %s", name)
	}

	// Reject path traversal
	if strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, "/..") {
		return "", fmt.Errorf("archive contains path traversal: %s", name)
	}

	// Verify the resolved path is within destination
	target := filepath.Join(destDir, cleaned)
	rel, err := filepath.Rel(destDir, target)
	if err != nil {
		return "", fmt.Errorf("resolving path %s: %w", name, err)
	}
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("archive path escapes destination: %s", name)
	}

	return cleaned, nil
}
