package registry

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Client struct {
	HTTPClient *http.Client
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
func (c *Client) DetectDefaultBranch(source *RepoSource) string {
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

	data, err := c.fetch(apiURL, source.Token, source.Username)
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

func (c *Client) FetchIndex(source *RepoSource) (*Index, error) {
	indexURL := source.IndexURL()

	data, err := c.fetch(indexURL, source.Token, source.Username)
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

func (c *Client) FetchAllIndexes(sources []RepoSource) (*Index, error) {
	var indexes []*Index

	for _, src := range sources {
		src := src
		idx, err := c.FetchIndex(&src)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to fetch from %s: %v\n", src.Name, err)
			continue
		}
		indexes = append(indexes, idx)
	}

	return MergeIndexes(indexes...), nil
}

func (c *Client) Download(url, dest, token, username string) error {
	data, err := c.fetch(url, token, username)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", url, err)
	}

	if err := os.WriteFile(dest, data, 0644); err != nil {
		return fmt.Errorf("writing to %s: %w", dest, err)
	}

	return nil
}

func (c *Client) fetch(url, token, username string) ([]byte, error) {
	if isLocalPath(url) {
		return os.ReadFile(url)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// GitHub Enterprise API: raw 파일 내용을 직접 받기 위한 Accept 헤더
	if strings.Contains(url, "/api/v3/repos/") && strings.Contains(url, "/contents/") {
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

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	credentialHint := "--token and --username"
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		if token == "" {
			return nil, fmt.Errorf("HTTP %d from %s (authentication required; use %s to provide credentials)", resp.StatusCode, url, credentialHint)
		}
		return nil, fmt.Errorf("HTTP %d from %s (credentials may be invalid or expired)", resp.StatusCode, url)
	}

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		if token == "" {
			return nil, fmt.Errorf("HTTP %d redirect from %s (private registry? use %s to provide credentials)", resp.StatusCode, url, credentialHint)
		}
		return nil, fmt.Errorf("HTTP %d redirect from %s", resp.StatusCode, url)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		if token == "" {
			return nil, fmt.Errorf("received HTML instead of JSON from %s (private registry? use %s to provide credentials)", url, credentialHint)
		}
		return nil, fmt.Errorf("received HTML instead of JSON from %s (credentials may be invalid or expired)", url)
	}

	return io.ReadAll(resp.Body)
}
