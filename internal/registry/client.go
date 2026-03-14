// Package registry handles communication with remote skill registries.
package registry

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	maxDownloadSize    = 500 * 1024 * 1024 // 500MB for archive downloads
	maxAPIResponseSize = 10 * 1024 * 1024  // 10MB for API/JSON responses
)

type Client struct {
	HTTPClient *http.Client
	OnProgress func(filename string) // called for each file downloaded in directory mode
}

func NewClient() *Client {
	return &Client{
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 0 && req.URL.Host != via[0].URL.Host {
					return http.ErrUseLastResponse
				}
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

// DetectDefaultBranch queries the GitHub API to find the default branch.
// For non-github.com hosts, it uses the /api/v3/ endpoint.
// Returns "main" as fallback if detection fails.
func (c *Client) DetectDefaultBranch(ctx context.Context, source *RepoSource) string {
	if isLocalPath(source.URL) {
		return "main"
	}

	u, err := url.Parse(source.URL)
	if err != nil {
		return "main"
	}
	ownerRepo := strings.Trim(u.Path, "/")

	var apiURL string
	if u.Host == "github.com" {
		apiURL = fmt.Sprintf("https://api.github.com/repos/%s", ownerRepo)
	} else {
		apiURL = fmt.Sprintf("%s://%s/api/v3/repos/%s", u.Scheme, u.Host, ownerRepo)
	}

	data, err := c.fetch(ctx, apiURL, source.Token, source.Username)
	if err != nil {
		return "main"
	}

	var repo struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.Unmarshal(data, &repo); err != nil || repo.DefaultBranch == "" {
		return "main"
	}
	return repo.DefaultBranch
}

func (c *Client) FetchIndex(ctx context.Context, source *RepoSource) (*Index, error) {
	indexURL := source.IndexURL()

	data, err := c.fetch(ctx, indexURL, source.Token, source.Username)
	if err != nil {
		return nil, fmt.Errorf("fetching index from %s: %w", source.Name, err)
	}

	idx, err := ParseIndex(data)
	if err != nil {
		return nil, fmt.Errorf("parsing index from %s: %w", source.Name, err)
	}

	for i := range idx.Skills {
		idx.Skills[i].Registry = source.Name
	}

	return idx, nil
}

func (c *Client) FetchAllIndexes(ctx context.Context, sources []RepoSource) (*Index, error) {
	var indexes []*Index

	for _, src := range sources {
		idx, err := c.FetchIndex(ctx, &src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to fetch from %s: %v\n", src.Name, err)
			continue
		}
		indexes = append(indexes, idx)
	}

	return MergeIndexes(indexes...), nil
}

func (c *Client) Download(ctx context.Context, rawURL, dest, token, username string) error {
	if isLocalPath(rawURL) {
		data, err := os.ReadFile(rawURL)
		if err != nil {
			return fmt.Errorf("reading local file %s: %w", rawURL, err)
		}
		return os.WriteFile(dest, data, 0644)
	}

	req, err := c.newRequest(ctx, rawURL, token, username)
	if err != nil {
		return fmt.Errorf("creating request for %s: %w", rawURL, err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp, rawURL, token); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(dest), ".download-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	n, err := io.Copy(tmpFile, io.LimitReader(resp.Body, maxDownloadSize+1))
	tmpFile.Close()
	if err != nil {
		return fmt.Errorf("writing download to %s: %w", dest, err)
	}
	if n > maxDownloadSize {
		return fmt.Errorf("download from %s exceeds maximum size (%d bytes)", rawURL, maxDownloadSize)
	}

	if err := os.Rename(tmpPath, dest); err != nil {
		return fmt.Errorf("moving download to %s: %w", dest, err)
	}

	return nil
}

// githubContentEntry represents a file or directory from the GitHub Contents API.
type githubContentEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"` // "file" or "dir"
}

// DownloadDirectory downloads all files from a directory source into destDir.
// For local paths, it copies the directory tree. For GitHub, it uses the Contents API.
// If OnProgress is set, it is called with each file's relative name after download.
func (c *Client) DownloadDirectory(ctx context.Context, source *RepoSource, dirPath, destDir string) error {
	dirPath = strings.TrimSuffix(dirPath, "/")

	if isLocalPath(source.URL) {
		srcDir := filepath.Join(source.URL, dirPath)
		return c.copyDirectory(srcDir, destDir)
	}

	return c.downloadGitHubDirectory(ctx, source, dirPath, destDir)
}

func (c *Client) copyDirectory(srcDir, destDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files/directories (matching archive mode behavior)
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

		destPath := filepath.Join(destDir, rel)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return err
		}
		if c.OnProgress != nil {
			c.OnProgress(rel)
		}
		return nil
	})
}

// maxConcurrentDownloads limits the number of concurrent file downloads.
const maxConcurrentDownloads = 8

func (c *Client) downloadGitHubDirectory(ctx context.Context, source *RepoSource, dirPath, destDir string) error {
	apiURL := source.ContentsAPIURL(dirPath)

	data, err := c.fetch(ctx, apiURL, source.Token, source.Username)
	if err != nil {
		return fmt.Errorf("listing directory %s: %w", dirPath, err)
	}

	var entries []githubContentEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("parsing directory listing for %s: %w", dirPath, err)
	}

	// Separate directories and files
	var dirs, files []githubContentEntry
	for _, entry := range entries {
		switch entry.Type {
		case "dir":
			dirs = append(dirs, entry)
		case "file":
			files = append(files, entry)
		}
	}

	// Create directories and recurse (sequential — dirs must exist before files)
	for _, entry := range dirs {
		destPath := filepath.Join(destDir, entry.Name)
		if err := os.MkdirAll(destPath, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", destPath, err)
		}
		if err := c.downloadGitHubDirectory(ctx, source, entry.Path, destPath); err != nil {
			return err
		}
	}

	// Download files concurrently with bounded worker pool
	sem := make(chan struct{}, maxConcurrentDownloads)
	var mu sync.Mutex
	var firstErr error
	var wg sync.WaitGroup

	for _, entry := range files {
		wg.Add(1)
		go func(entry githubContentEntry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			mu.Lock()
			if firstErr != nil {
				mu.Unlock()
				return
			}
			mu.Unlock()

			destPath := filepath.Join(destDir, entry.Name)
			downloadURL := source.ResolveDownloadURL(entry.Path)
			if err := c.Download(ctx, downloadURL, destPath, source.Token, source.Username); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("downloading %s: %w", entry.Name, err)
				}
				mu.Unlock()
				return
			}
			if c.OnProgress != nil {
				c.OnProgress(entry.Name)
			}
		}(entry)
	}

	wg.Wait()
	return firstErr
}

func (c *Client) newRequest(ctx context.Context, rawURL, token, username string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	if strings.Contains(rawURL, "/api/v3/repos/") && strings.Contains(rawURL, "/contents/") {
		req.Header.Set("Accept", "application/vnd.github.raw")
	}
	if token != "" {
		if username != "" {
			encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + token))
			req.Header.Set("Authorization", "Basic "+encoded)
		} else {
			req.Header.Set("Authorization", "token "+token)
		}
	}
	return req, nil
}

func checkResponse(resp *http.Response, rawURL, token string) error {
	credentialHint := "--token and --username"

	// Check for GitHub API rate limiting
	if resp.StatusCode == http.StatusForbidden && resp.Header.Get("X-RateLimit-Remaining") == "0" {
		if token == "" {
			return fmt.Errorf("GitHub API rate limit exceeded (60/hr for unauthenticated requests). Use 'skillhub repo add <url> --token <PAT>' to increase to 5000/hr")
		}
		return fmt.Errorf("GitHub API rate limit exceeded. Wait or use a different token")
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		if token == "" {
			return fmt.Errorf("HTTP %d from %s (authentication required; use %s to provide credentials)", resp.StatusCode, rawURL, credentialHint)
		}
		return fmt.Errorf("HTTP %d from %s (credentials may be invalid or expired)", resp.StatusCode, rawURL)
	}
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		if token == "" {
			return fmt.Errorf("HTTP %d redirect from %s (private registry? use %s to provide credentials)", resp.StatusCode, rawURL, credentialHint)
		}
		return fmt.Errorf("HTTP %d redirect from %s", resp.StatusCode, rawURL)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, rawURL)
	}
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		if token == "" {
			return fmt.Errorf("received HTML instead of JSON from %s (private registry? use %s to provide credentials)", rawURL, credentialHint)
		}
		return fmt.Errorf("received HTML instead of JSON from %s (credentials may be invalid or expired)", rawURL)
	}
	return nil
}

func (c *Client) fetch(ctx context.Context, rawURL, token, username string) ([]byte, error) {
	if isLocalPath(rawURL) {
		return os.ReadFile(rawURL)
	}

	req, err := c.newRequest(ctx, rawURL, token, username)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := checkResponse(resp, rawURL, token); err != nil {
		return nil, err
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxAPIResponseSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxAPIResponseSize {
		return nil, fmt.Errorf("response from %s exceeds maximum size (%d bytes)", rawURL, maxAPIResponseSize)
	}

	return data, nil
}
