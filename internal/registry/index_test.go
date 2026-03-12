package registry

import "testing"

func TestParseIndex(t *testing.T) {
	data := []byte(`{
		"skills": [
			{
				"name": "code-review",
				"version": "1.0.0",
				"description": "Review code changes",
				"tags": ["engineering"],
				"download_url": "packages/code-review-1.0.0.tar.gz",
				"checksum": "sha256:abc123"
			},
			{
				"name": "pr-summary",
				"version": "0.9.1",
				"description": "Summarize pull requests",
				"tags": ["collaboration"],
				"download_url": "packages/pr-summary-0.9.1.tar.gz"
			}
		]
	}`)

	idx, err := ParseIndex(data)
	if err != nil {
		t.Fatalf("ParseIndex: %v", err)
	}

	if len(idx.Skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(idx.Skills))
	}

	if idx.Skills[0].Name != "code-review" {
		t.Errorf("expected 'code-review', got %q", idx.Skills[0].Name)
	}
}

func TestParseIndexInvalid(t *testing.T) {
	_, err := ParseIndex([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSearch(t *testing.T) {
	idx := &Index{
		Skills: []IndexEntry{
			{Name: "code-review", Description: "Review code changes", Tags: []string{"engineering"}},
			{Name: "pr-summary", Description: "Summarize pull requests", Tags: []string{"collaboration"}},
			{Name: "architecture-check", Description: "Review architecture docs", Tags: []string{"engineering"}},
		},
	}

	results := idx.Search("review")
	if len(results) != 2 {
		t.Errorf("expected 2 results for 'review', got %d", len(results))
	}

	results = idx.Search("summary")
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'summary', got %d", len(results))
	}

	results = idx.Search("collaboration")
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'collaboration' tag, got %d", len(results))
	}

	results = idx.Search("nonexistent")
	if len(results) != 0 {
		t.Errorf("expected 0 results for 'nonexistent', got %d", len(results))
	}
}

func TestFind(t *testing.T) {
	idx := &Index{
		Skills: []IndexEntry{
			{Name: "code-review", Version: "1.0.0"},
			{Name: "pr-summary", Version: "0.9.1"},
		},
	}

	entry := idx.Find("code-review")
	if entry == nil {
		t.Fatal("expected to find 'code-review'")
	}
	if entry.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", entry.Version)
	}

	entry = idx.Find("nonexistent")
	if entry != nil {
		t.Error("expected nil for nonexistent skill")
	}
}

func TestMergeIndexes(t *testing.T) {
	idx1 := &Index{Skills: []IndexEntry{{Name: "a"}}}
	idx2 := &Index{Skills: []IndexEntry{{Name: "b"}, {Name: "c"}}}

	merged := MergeIndexes(idx1, idx2, nil)
	if len(merged.Skills) != 3 {
		t.Errorf("expected 3 skills, got %d", len(merged.Skills))
	}
}
