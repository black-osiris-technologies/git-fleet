package semver

import "testing"

func TestParseTag(t *testing.T) {
	cases := []struct {
		tag  string
		want Version
		ok   bool
	}{
		{"2.3.5", Version{2, 3, 5}, true},
		{"v2.3.5", Version{2, 3, 5}, true},
		{"  v0.1.0 ", Version{0, 1, 0}, true},
		{"2.4.0-rc.1", Version{}, false},
		{"v2.4", Version{}, false},
		{"release-2.4", Version{}, false},
		{"latest", Version{}, false},
	}
	for _, tc := range cases {
		got, ok := ParseTag(tc.tag)
		if ok != tc.ok || got != tc.want {
			t.Errorf("ParseTag(%q) = %v, %t; want %v, %t", tc.tag, got, ok, tc.want, tc.ok)
		}
	}
}

func TestParseVersion(t *testing.T) {
	got, err := ParseVersion("2.4")
	if err != nil {
		t.Fatalf("ParseVersion(2.4) error = %v", err)
	}
	if got != (Version{2, 4, 0}) {
		t.Fatalf("ParseVersion(2.4) = %v, want 2.4.0", got)
	}

	got, err = ParseVersion("v3.1.7")
	if err != nil {
		t.Fatalf("ParseVersion(v3.1.7) error = %v", err)
	}
	if got != (Version{3, 1, 7}) {
		t.Fatalf("ParseVersion(v3.1.7) = %v, want 3.1.7", got)
	}

	if _, err := ParseVersion("not-a-version"); err == nil {
		t.Fatal("ParseVersion(not-a-version) error = nil, want error")
	}
}

func TestHighestIgnoresPrereleaseAndJunk(t *testing.T) {
	// 1.10.0 must beat 1.9.0 (numeric, not lexicographic) and the pre-release
	// and non-semver tags must be ignored entirely.
	got, ok := Highest([]string{"v1.9.0", "v1.10.0", "v1.10.1-rc.1", "nightly"})
	if !ok {
		t.Fatal("Highest() ok = false, want true")
	}
	if got != (Version{1, 10, 0}) {
		t.Fatalf("Highest() = %v, want 1.10.0", got)
	}
}

func TestHighestNoEligibleTags(t *testing.T) {
	if _, ok := Highest([]string{"nightly", "2.4.0-rc.1"}); ok {
		t.Fatal("Highest() ok = true, want false when no stable tag present")
	}
}

func TestNext(t *testing.T) {
	minor, err := Next(Version{2, 3, 5}, BumpMinor)
	if err != nil {
		t.Fatalf("Next(minor) error = %v", err)
	}
	if minor != (Version{2, 4, 0}) {
		t.Fatalf("Next(2.3.5, minor) = %v, want 2.4.0", minor)
	}

	major, err := Next(Version{2, 3, 5}, BumpMajor)
	if err != nil {
		t.Fatalf("Next(major) error = %v", err)
	}
	if major != (Version{3, 0, 0}) {
		t.Fatalf("Next(2.3.5, major) = %v, want 3.0.0", major)
	}

	if _, err := Next(Version{1, 0, 0}, Bump("patch")); err == nil {
		t.Fatal("Next(patch) error = nil, want error for unsupported bump")
	}
}
