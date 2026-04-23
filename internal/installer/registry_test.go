package installer

import (
	"strings"
	"testing"
)

func TestLoad_EmbeddedYAML(t *testing.T) {
	reg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(reg.Scanners) == 0 {
		t.Fatal("expected at least one scanner in the pin DB")
	}
	for _, want := range []string{"gitleaks", "opengrep", "osv-scanner", "biome", "ruff", "golangci-lint", "shellcheck"} {
		if _, ok := reg.Scanners[want]; !ok {
			t.Errorf("pin DB missing scanner %q", want)
		}
	}
}

func TestLookup_UnknownScanner(t *testing.T) {
	reg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = reg.Lookup("bogus", "1.0.0", "linux_amd64")
	if err == nil || !strings.Contains(err.Error(), "not in the pin database") {
		t.Fatalf("expected unknown-scanner error, got %v", err)
	}
}

func TestLookup_UnknownVersion(t *testing.T) {
	reg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = reg.Lookup("gitleaks", "99.99.99", "linux_amd64")
	if err == nil || !strings.Contains(err.Error(), "no pinned entry for version") {
		t.Fatalf("expected unknown-version error, got %v", err)
	}
}

func TestLookup_UnknownPlatform(t *testing.T) {
	reg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = reg.Lookup("gitleaks", "8.28.0", "plan9_mips")
	if err == nil || !strings.Contains(err.Error(), "no asset for platform") {
		t.Fatalf("expected unknown-platform error, got %v", err)
	}
}

func TestLookup_ShippedPinsArePopulated(t *testing.T) {
	// Regression guard: the release-engineering script (scripts/refresh-pins.go)
	// must have filled every sha256 before release. A placeholder slipping
	// through would silently fail at install time with a confusing error.
	reg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for scannerName, sc := range reg.Scanners {
		for version, ve := range sc.Versions {
			for platform, pa := range ve.Platforms {
				if isPlaceholderHash(pa.ArchiveSHA256) || isPlaceholderHash(pa.BinarySHA256) {
					t.Errorf("placeholder hash still present: %s@%s/%s — run `go run ./scripts/refresh-pins.go`",
						scannerName, version, platform)
				}
			}
		}
	}
}

func TestLookup_PlaceholderHashIsRejected(t *testing.T) {
	// Exercises the placeholder-rejection path via a synthetic registry so
	// the check stays covered even though the shipped pins are populated.
	reg := &Registry{
		Scanners: map[string]Scanner{
			"fake": {
				Versions: map[string]VersionEntry{
					"0.0.0": {
						Archive: ArchiveRaw,
						Binary:  "fake",
						Platforms: map[string]PlatformAsset{
							"linux_amd64": {
								URL:           "https://github.com/example/fake/releases/download/v0.0.0/fake",
								ArchiveSHA256: "0000000000000000000000000000000000000000000000000000000000000000",
								BinarySHA256:  "0000000000000000000000000000000000000000000000000000000000000000",
							},
						},
					},
				},
			},
		},
	}
	_, _, err := reg.Lookup("fake", "0.0.0", "linux_amd64")
	if err == nil {
		t.Fatal("expected placeholder-hash rejection, got nil")
	}
	if !strings.Contains(err.Error(), "placeholder sha256") {
		t.Fatalf("expected placeholder error, got %v", err)
	}
}

func TestIsPlaceholderHash(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", true},
		{"0000000000000000000000000000000000000000000000000000000000000000", true},
		{"abc", true},
		{"abc1234567890abc1234567890abc1234567890abc1234567890abc123456789", false}, // 64 hex, not all zero
	}
	for _, c := range cases {
		if got := isPlaceholderHash(c.in); got != c.want {
			t.Errorf("isPlaceholderHash(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
