package registry

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

type RepoSource struct {
	Name     string
	URL      string
	Token    string
	Username string
	Branch   string
}

func (r *RepoSource) branch() string {
	if r.Branch != "" {
		return r.Branch
	}
	return "main"
}

func (r *RepoSource) IndexURL() string {
	if isLocalPath(r.URL) {
		return filepath.Join(r.URL, "index.json")
	}
	return rawContentURL(r.URL, "index.json", r.branch())
}

func (r *RepoSource) ResolveDownloadURL(relative string) string {
	if strings.HasPrefix(relative, "http://") || strings.HasPrefix(relative, "https://") {
		return relative
	}
	if isLocalPath(r.URL) {
		return filepath.Join(r.URL, relative)
	}
	return rawContentURL(r.URL, relative, r.branch())
}

func ParseRepoURL(rawURL string) (*RepoSource, error) {
	raw := strings.TrimSpace(rawURL)

	if isLocalPath(raw) {
		name := filepath.Base(raw)
		return &RepoSource{Name: name, URL: raw}, nil
	}

	raw = strings.TrimSuffix(raw, ".git")
	raw = strings.TrimSuffix(raw, "/")

	// Handle shorthand: org/repo (defaults to github.com)
	if !strings.Contains(raw, "://") && strings.Count(raw, "/") == 1 {
		name := strings.Split(raw, "/")[1]
		return &RepoSource{Name: name, URL: "https://github.com/" + raw}, nil
	}

	// Handle any HTTP(S) URL (GitHub, GitHub Enterprise, Gitea, GitLab, etc.)
	if strings.HasPrefix(raw, "https://") || strings.HasPrefix(raw, "http://") {
		u, err := url.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %s", rawURL)
		}
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid repository URL (expected host/owner/repo): %s", rawURL)
		}
		name := parts[len(parts)-1]
		return &RepoSource{Name: name, URL: raw}, nil
	}

	return nil, fmt.Errorf("unsupported registry URL format: %s", rawURL)
}

func isLocalPath(url string) bool {
	return strings.HasPrefix(url, "/") || strings.HasPrefix(url, "./") || strings.HasPrefix(url, "../")
}

// IsLocal reports whether this source points to a local filesystem path.
func (r *RepoSource) IsLocal() bool {
	return isLocalPath(r.URL)
}

// ContentsAPIPutURL returns the GitHub Contents API URL for writing a file.
// Unlike ContentsAPIURL, it omits the ?ref= query parameter since the
// PUT endpoint accepts the branch in the JSON request body.
func (r *RepoSource) ContentsAPIPutURL(path string) string {
	path = strings.TrimSuffix(path, "/")
	if isLocalPath(r.URL) {
		return filepath.Join(r.URL, path)
	}

	u, err := url.Parse(r.URL)
	if err != nil {
		return ""
	}
	ownerRepo := strings.Trim(u.Path, "/")

	if u.Host == "github.com" {
		return fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", ownerRepo, path)
	}
	return fmt.Sprintf("%s://%s/api/v3/repos/%s/contents/%s", u.Scheme, u.Host, ownerRepo, path)
}

// ContentsAPIURL returns the GitHub Contents API URL for the given path.
// For local paths, it returns the joined filesystem path.
func (r *RepoSource) ContentsAPIURL(path string) string {
	path = strings.TrimSuffix(path, "/")
	if isLocalPath(r.URL) {
		return filepath.Join(r.URL, path)
	}

	u, err := url.Parse(r.URL)
	if err != nil {
		return ""
	}
	ownerRepo := strings.Trim(u.Path, "/")

	if u.Host == "github.com" {
		return fmt.Sprintf("https://api.github.com/repos/%s/contents/%s?ref=%s", ownerRepo, path, r.branch())
	}
	// GitHub Enterprise
	return fmt.Sprintf("%s://%s/api/v3/repos/%s/contents/%s?ref=%s", u.Scheme, u.Host, ownerRepo, path, r.branch())
}

func rawContentURL(repoURL, path, branch string) string {
	// GitHub.com: use raw.githubusercontent.com
	if strings.HasPrefix(repoURL, "https://github.com/") {
		trimmed := strings.TrimPrefix(repoURL, "https://github.com/")
		return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", trimmed, branch, path)
	}
	// GitHub Enterprise / 기타: REST API 엔드포인트 사용
	u, err := url.Parse(repoURL)
	if err != nil {
		return fmt.Sprintf("%s/raw/%s/%s", repoURL, branch, path)
	}
	ownerRepo := strings.Trim(u.Path, "/")
	return fmt.Sprintf("%s://%s/api/v3/repos/%s/contents/%s?ref=%s", u.Scheme, u.Host, ownerRepo, path, branch)
}
