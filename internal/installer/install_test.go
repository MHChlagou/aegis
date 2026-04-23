package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newFixtureServer serves a fixed byte slice at / over HTTPS (httptest.TLS)
// and returns the server plus the sha256 of what it's serving.
func newFixtureServer(t *testing.T, payload []byte) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	t.Cleanup(srv.Close)
	sum := sha256.Sum256(payload)
	return srv, hex.EncodeToString(sum[:])
}

// testRegistry builds a single-entry registry backed by the given URL +
// hashes. For raw archives the archive/binary hashes are identical; for
// the tests here we only exercise ArchiveRaw so the caller passes one.
// Using a constructor keeps tests independent of the embedded scanners.yaml.
func testRegistry(archive ArchiveType, binary, url, archiveSHA, binarySHA string) *Registry {
	return &Registry{
		Scanners: map[string]Scanner{
			"fakesc": {
				Versions: map[string]VersionEntry{
					"1.0.0": {
						Archive: archive,
						Binary:  binary,
						Platforms: map[string]PlatformAsset{
							"linux_amd64": {
								URL:           url,
								ArchiveSHA256: archiveSHA,
								BinarySHA256:  binarySHA,
							},
						},
					},
				},
			},
		},
	}
}

func TestInstall_HappyPath_RawBinary(t *testing.T) {
	payload := []byte(fakeBinaryPayload)
	srv, want := newFixtureServer(t, payload)
	// For raw archives, archive bytes == binary bytes, so both pins match.
	reg := testRegistry(ArchiveRaw, "fakesc", srv.URL, want, want)

	dest := t.TempDir()
	res, err := Install(reg, Options{
		Scanner:      "fakesc",
		Version:      "1.0.0",
		Platform:     "linux_amd64",
		DestDir:      dest,
		HTTP:         srv.Client(),
		AllowedHosts: map[string]bool{"127.0.0.1": true},
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if res.ArchiveSHA256 != want || res.BinarySHA256 != want {
		t.Errorf("hash mismatch: archive=%s binary=%s want=%s", res.ArchiveSHA256, res.BinarySHA256, want)
	}
	assertFileMatches(t, filepath.Join(dest, "fakesc"), fakeBinaryPayload)
	assertExecutable(t, filepath.Join(dest, "fakesc"))
}

func TestInstall_HashMismatch_Refuses(t *testing.T) {
	payload := []byte(fakeBinaryPayload)
	srv, _ := newFixtureServer(t, payload)
	// Wrong archive pin — install must refuse before writing anything.
	reg := testRegistry(ArchiveRaw, "fakesc", srv.URL,
		"1111111111111111111111111111111111111111111111111111111111111111",
		"1111111111111111111111111111111111111111111111111111111111111111")

	dest := t.TempDir()
	_, err := Install(reg, Options{
		Scanner:      "fakesc",
		Version:      "1.0.0",
		Platform:     "linux_amd64",
		DestDir:      dest,
		HTTP:         srv.Client(),
		AllowedHosts: map[string]bool{"127.0.0.1": true},
	})
	if err == nil {
		t.Fatal("expected hash-mismatch error, got nil")
	}
	if !strings.Contains(err.Error(), "archive sha256 mismatch") {
		t.Fatalf("expected archive sha256 mismatch, got: %v", err)
	}
	if _, serr := os.Stat(filepath.Join(dest, "fakesc")); serr == nil {
		t.Error("binary was written despite hash mismatch")
	}
}

func TestInstall_HostNotOnAllowlist_Refuses(t *testing.T) {
	payload := []byte(fakeBinaryPayload)
	srv, want := newFixtureServer(t, payload)
	reg := testRegistry(ArchiveRaw, "fakesc", srv.URL, want, want)
	_, err := Install(reg, Options{
		Scanner:  "fakesc",
		Version:  "1.0.0",
		Platform: "linux_amd64",
		DestDir:  t.TempDir(),
		HTTP:     srv.Client(),
		// AllowedHosts intentionally omitted — should fall back to the
		// package default, which does NOT include 127.0.0.1.
	})
	if err == nil {
		t.Fatal("expected allowlist rejection")
	}
	if !strings.Contains(err.Error(), "not on the allowlist") {
		t.Fatalf("expected allowlist error, got: %v", err)
	}
}

func TestInstall_NonHTTPS_Refuses(t *testing.T) {
	reg := testRegistry(ArchiveRaw, "fakesc",
		"http://example.com/scanner",
		"1111111111111111111111111111111111111111111111111111111111111111",
		"1111111111111111111111111111111111111111111111111111111111111111")
	_, err := Install(reg, Options{
		Scanner:      "fakesc",
		Version:      "1.0.0",
		Platform:     "linux_amd64",
		DestDir:      t.TempDir(),
		AllowedHosts: map[string]bool{"example.com": true}, // host is allowed, but scheme check should still refuse
	})
	if err == nil {
		t.Fatal("expected refusal for non-HTTPS URL")
	}
	// The refusal may come from either the host allowlist (if implemented)
	// or the downloader's scheme guard. Either is acceptable.
	if !strings.Contains(err.Error(), "HTTPS") && !strings.Contains(err.Error(), "https") {
		t.Fatalf("expected HTTPS refusal, got: %v", err)
	}
}
