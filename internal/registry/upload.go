package registry

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type githubPutRequest struct {
	Message string `json:"message"`
	Content string `json:"content"`
	Branch  string `json:"branch"`
	SHA     string `json:"sha,omitempty"`
}

type githubFileInfo struct {
	SHA string `json:"sha"`
}

// GetFileSHA returns the blob SHA of a file in a GitHub repository.
// Returns "" if the file does not exist (404).
func (c *Client) GetFileSHA(ctx context.Context, source *RepoSource, path string) (string, error) {
	if source.IsLocal() {
		return "", nil
	}

	apiURL := source.ContentsAPIPutURL(path) + "?ref=" + source.branch()

	req, err := c.newRequest(ctx, apiURL, source.Token, source.Username)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.object+json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching file info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, path)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxAPIResponseSize))
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	var info githubFileInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return "", fmt.Errorf("parsing file info: %w", err)
	}
	return info.SHA, nil
}

// UploadFile uploads content to a registry. For local registries it writes
// directly to the filesystem. For GitHub it uses the Contents API.
func (c *Client) UploadFile(ctx context.Context, source *RepoSource, path string, content []byte, sha, message string) error {
	if source.IsLocal() {
		dest := filepath.Join(source.URL, path)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}
		return os.WriteFile(dest, content, 0644)
	}

	apiURL := source.ContentsAPIPutURL(path)

	body := githubPutRequest{
		Message: message,
		Content: base64.StdEncoding.EncodeToString(content),
		Branch:  source.branch(),
		SHA:     sha,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")

	if source.Token != "" {
		if source.Username != "" {
			encoded := base64.StdEncoding.EncodeToString([]byte(source.Username + ":" + source.Token))
			req.Header.Set("Authorization", "Basic "+encoded)
		} else {
			req.Header.Set("Authorization", "token "+source.Token)
		}
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("uploading %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("HTTP %d uploading %s: %s", resp.StatusCode, path, string(respBody))
	}

	return nil
}

// UploadDirectory uploads all files from srcDir to destPrefix in the registry.
// For local registries it copies the directory tree. For GitHub it uploads
// each file via the Contents API. Hidden files/directories are skipped.
func (c *Client) UploadDirectory(ctx context.Context, source *RepoSource, srcDir, destPrefix, message string) error {
	if source.IsLocal() {
		destDir := filepath.Join(source.URL, destPrefix)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", destDir, err)
		}
		return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Name() != "." && len(d.Name()) > 0 && d.Name()[0] == '.' {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			rel, err := filepath.Rel(srcDir, path)
			if err != nil {
				return err
			}
			dest := filepath.Join(destDir, rel)
			if d.IsDir() {
				return os.MkdirAll(dest, 0755)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("reading %s: %w", path, err)
			}
			return os.WriteFile(dest, data, 0644)
		})
	}

	// GitHub: walk source directory and upload each file
	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name() != "." && len(d.Name()) > 0 && d.Name()[0] == '.' {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		remotePath := destPrefix + "/" + filepath.ToSlash(rel)

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", rel, err)
		}

		sha, err := c.GetFileSHA(ctx, source, remotePath)
		if err != nil {
			return fmt.Errorf("checking %s: %w", remotePath, err)
		}

		if err := c.UploadFile(ctx, source, remotePath, content, sha, message); err != nil {
			return fmt.Errorf("uploading %s: %w", rel, err)
		}

		if c.OnProgress != nil {
			c.OnProgress(rel)
		}
		return nil
	})
}

// UpdateIndex fetches the current index.json from the registry, upserts the
// given entry, and writes it back. If force is false and a matching
// name+version already exists, it returns an error.
func (c *Client) UpdateIndex(ctx context.Context, source *RepoSource, entry IndexEntry, force bool) error {
	idx := &Index{}

	if source.IsLocal() {
		indexPath := filepath.Join(source.URL, "index.json")
		data, err := os.ReadFile(indexPath)
		if err == nil {
			parsed, parseErr := ParseIndex(data)
			if parseErr == nil {
				idx = parsed
			}
		}
	} else {
		data, err := c.fetch(ctx, source.IndexURL(), source.Token, source.Username)
		if err == nil {
			parsed, parseErr := ParseIndex(data)
			if parseErr == nil {
				idx = parsed
			}
		}
	}

	// Check for existing version
	if !force {
		if existing := idx.FindVersion(entry.Name, entry.Version); existing != nil {
			return fmt.Errorf("version %s@%s already exists in %s (use --force to overwrite)", entry.Name, entry.Version, source.Name)
		}
	}

	// Remove existing entry with same name+version (for force overwrite)
	var filtered []IndexEntry
	for _, e := range idx.Skills {
		if e.Name == entry.Name && e.Version == entry.Version {
			continue
		}
		filtered = append(filtered, e)
	}
	filtered = append(filtered, entry)
	idx.Skills = filtered

	indexData, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling index: %w", err)
	}
	indexData = append(indexData, '\n')

	if source.IsLocal() {
		return os.WriteFile(filepath.Join(source.URL, "index.json"), indexData, 0644)
	}

	sha, err := c.GetFileSHA(ctx, source, "index.json")
	if err != nil {
		return fmt.Errorf("getting index.json SHA: %w", err)
	}

	msg := fmt.Sprintf("update index: %s@%s", entry.Name, entry.Version)
	return c.UploadFile(ctx, source, "index.json", indexData, sha, msg)
}
