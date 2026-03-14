package skill

import (
	"fmt"
	"strconv"
	"strings"
)

// SemVer represents a semantic version with major, minor, and patch components.
type SemVer struct {
	Major int
	Minor int
	Patch int
}

// String returns the semver string representation "Major.Minor.Patch".
func (v SemVer) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// ParseVersion parses a "X.Y.Z" version string into a SemVer.
func ParseVersion(s string) (SemVer, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return SemVer{}, fmt.Errorf("invalid version format %q: expected X.Y.Z", s)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid major version %q: %w", parts[0], err)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid minor version %q: %w", parts[1], err)
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid patch version %q: %w", parts[2], err)
	}

	return SemVer{Major: major, Minor: minor, Patch: patch}, nil
}

// CompareVersions returns -1 if a < b, 0 if a == b, 1 if a > b.
func CompareVersions(a, b string) (int, error) {
	va, err := ParseVersion(a)
	if err != nil {
		return 0, err
	}

	vb, err := ParseVersion(b)
	if err != nil {
		return 0, err
	}

	if va.Major != vb.Major {
		if va.Major < vb.Major {
			return -1, nil
		}
		return 1, nil
	}

	if va.Minor != vb.Minor {
		if va.Minor < vb.Minor {
			return -1, nil
		}
		return 1, nil
	}

	if va.Patch != vb.Patch {
		if va.Patch < vb.Patch {
			return -1, nil
		}
		return 1, nil
	}

	return 0, nil
}
