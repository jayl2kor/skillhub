package skill

import "testing"

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input   string
		want    SemVer
		wantErr bool
	}{
		{"1.0.0", SemVer{1, 0, 0}, false},
		{"0.9.1", SemVer{0, 9, 1}, false},
		{"10.20.30", SemVer{10, 20, 30}, false},
		{"", SemVer{}, true},
		{"1.0", SemVer{}, true},
		{"a.b.c", SemVer{}, true},
		{"1.0.0.0", SemVer{}, true},
	}

	for _, tt := range tests {
		got, err := ParseVersion(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseVersion(%q): wantErr=%v, got %v", tt.input, tt.wantErr, err)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("ParseVersion(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
		{"1.1.0", "1.0.0", 1},
		{"1.0.0", "1.1.0", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"0.9.1", "1.0.0", -1},
		{"10.0.0", "9.0.0", 1},
	}

	for _, tt := range tests {
		got, err := CompareVersions(tt.a, tt.b)
		if err != nil {
			t.Errorf("CompareVersions(%q, %q): unexpected error: %v", tt.a, tt.b, err)
			continue
		}
		if got != tt.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestCompareVersionsInvalid(t *testing.T) {
	_, err := CompareVersions("bad", "1.0.0")
	if err == nil {
		t.Error("expected error for invalid version")
	}

	_, err = CompareVersions("1.0.0", "bad")
	if err == nil {
		t.Error("expected error for invalid version")
	}
}
