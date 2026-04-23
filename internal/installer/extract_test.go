package installer

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ulikunitz/xz"
)

const fakeBinaryPayload = "#!/bin/sh\necho hello-from-fake-scanner\n"

func TestExtract_Raw(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "raw-binary")
	if err := os.WriteFile(src, []byte(fakeBinaryPayload), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "out", "scanner")
	if err := ExtractBinary(src, ArchiveRaw, "", dest); err != nil {
		t.Fatalf("ExtractBinary: %v", err)
	}
	assertFileMatches(t, dest, fakeBinaryPayload)
	assertExecutable(t, dest)
}

func TestExtract_TarGz(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "bundle.tar.gz")
	buildTarGz(t, src, map[string]string{
		"bundle-1.0.0/README":  "docs",
		"bundle-1.0.0/scanner": fakeBinaryPayload,
	})
	dest := filepath.Join(dir, "out", "scanner")
	if err := ExtractBinary(src, ArchiveTarGz, "scanner", dest); err != nil {
		t.Fatalf("ExtractBinary: %v", err)
	}
	assertFileMatches(t, dest, fakeBinaryPayload)
	assertExecutable(t, dest)
}

func TestExtract_TarXz(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "bundle.tar.xz")
	buildTarXz(t, src, map[string]string{
		"scanner": fakeBinaryPayload,
	})
	dest := filepath.Join(dir, "out", "scanner")
	if err := ExtractBinary(src, ArchiveTarXz, "scanner", dest); err != nil {
		t.Fatalf("ExtractBinary: %v", err)
	}
	assertFileMatches(t, dest, fakeBinaryPayload)
}

func TestExtract_Zip(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "bundle.zip")
	buildZip(t, src, map[string]string{
		"nested/scanner": fakeBinaryPayload,
	})
	dest := filepath.Join(dir, "out", "scanner")
	if err := ExtractBinary(src, ArchiveZip, "scanner", dest); err != nil {
		t.Fatalf("ExtractBinary: %v", err)
	}
	assertFileMatches(t, dest, fakeBinaryPayload)
}

func TestExtract_BinaryNotFoundInArchive(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "bundle.tar.gz")
	buildTarGz(t, src, map[string]string{"bundle/other": "nope"})
	err := ExtractBinary(src, ArchiveTarGz, "scanner", filepath.Join(dir, "out", "scanner"))
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

func buildTarGz(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	writeTarEntries(t, tw, files)
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
}

func buildTarXz(t *testing.T, path string, files map[string]string) {
	t.Helper()
	// Build tar in memory, then xz-compress into the file.
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	writeTarEntries(t, tw, files)
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	xw, err := xz.NewWriter(f)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(xw, &buf); err != nil {
		t.Fatal(err)
	}
	if err := xw.Close(); err != nil {
		t.Fatal(err)
	}
}

func buildZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	zw := zip.NewWriter(f)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeTarEntries(t *testing.T, tw *tar.Writer, files map[string]string) {
	t.Helper()
	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0o755,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
}

func assertFileMatches(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != want {
		t.Errorf("content mismatch at %s:\n  got:  %q\n  want: %q", path, got, want)
	}
}

func assertExecutable(t *testing.T, path string) {
	t.Helper()
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0o100 == 0 {
		t.Errorf("%s is not executable (mode=%v)", path, st.Mode())
	}
}
