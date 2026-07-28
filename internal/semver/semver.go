// Package semver provides the minimal semantic-version handling git-fleet needs
// to derive the next release line from a repository's existing tags. It is not a
// general-purpose SemVer implementation: it intentionally ignores build metadata
// and treats any pre-release tag (for example 2.4.0-rc.1) as not eligible to seed
// a new release line.
package semver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Version is a released MAJOR.MINOR.PATCH triple.
type Version struct {
	Major int
	Minor int
	Patch int
}

// Bump identifies which component release-start advances.
type Bump string

const (
	// BumpMinor starts the next minor line and resets the patch to zero.
	BumpMinor Bump = "minor"
	// BumpMajor starts the next major line and resets minor and patch to zero.
	BumpMajor Bump = "major"
)

var (
	// tagPattern matches a stable release tag with an optional leading "v".
	// A pre-release or build-metadata suffix (anything after the patch number)
	// makes the tag ineligible and is rejected here.
	tagPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)$`)
	// versionPattern accepts an explicit version passed on the command line,
	// where the patch component is optional (2.4 is treated as 2.4.0).
	versionPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)(?:\.(\d+))?$`)
)

// ParseTag parses a stable release tag such as "v2.3.5" or "2.3.5". It reports
// ok=false for pre-release tags, build metadata, or anything that is not a bare
// MAJOR.MINOR.PATCH triple, so callers can skip ineligible tags without error.
func ParseTag(tag string) (Version, bool) {
	matches := tagPattern.FindStringSubmatch(strings.TrimSpace(tag))
	if matches == nil {
		return Version{}, false
	}
	return Version{
		Major: mustAtoi(matches[1]),
		Minor: mustAtoi(matches[2]),
		Patch: mustAtoi(matches[3]),
	}, true
}

// ParseVersion parses an explicit --version argument. Unlike ParseTag it allows
// a two-part MAJOR.MINOR value and defaults the patch to zero.
func ParseVersion(value string) (Version, error) {
	matches := versionPattern.FindStringSubmatch(strings.TrimSpace(value))
	if matches == nil {
		return Version{}, fmt.Errorf("invalid version %q; expected MAJOR.MINOR or MAJOR.MINOR.PATCH", value)
	}
	version := Version{
		Major: mustAtoi(matches[1]),
		Minor: mustAtoi(matches[2]),
	}
	if matches[3] != "" {
		version.Patch = mustAtoi(matches[3])
	}
	return version, nil
}

// Highest returns the greatest eligible version among the given tags. It reports
// ok=false when no tag is an eligible stable release, which callers treat as
// "no prior release" rather than an error.
func Highest(tags []string) (Version, bool) {
	var best Version
	found := false
	for _, tag := range tags {
		version, ok := ParseTag(tag)
		if !ok {
			continue
		}
		if !found || version.Compare(best) > 0 {
			best = version
			found = true
		}
	}
	return best, found
}

// Next returns the version that seeds the next release line for the requested
// bump. A minor bump advances the minor component and resets the patch; a major
// bump advances the major component and resets minor and patch.
func Next(current Version, bump Bump) (Version, error) {
	switch bump {
	case BumpMinor:
		return Version{Major: current.Major, Minor: current.Minor + 1}, nil
	case BumpMajor:
		return Version{Major: current.Major + 1}, nil
	default:
		return Version{}, fmt.Errorf("unknown bump %q", bump)
	}
}

// Compare returns a negative number when v sorts before other, zero when they
// are equal, and a positive number when v sorts after other.
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		return v.Major - other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor - other.Minor
	}
	return v.Patch - other.Patch
}

// String renders the version as MAJOR.MINOR.PATCH.
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func mustAtoi(value string) int {
	// The regular expressions guarantee the captured groups are digit runs, so
	// a parse failure here would be a programming error rather than bad input.
	number, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Sprintf("semver: unexpected non-numeric component %q", value))
	}
	return number
}
