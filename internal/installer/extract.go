package installer

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
)

// ExtractBinary reads the archive at archivePath, pulls out the binary named
// wantBinary according to archiveType, and writes it to destPath with mode
// 0755. For ArchiveRaw, archivePath IS the binary — it is moved into place.
//
// Finding the binary inside an archive is lenient: we match the basename,
// not the full internal path, because upstream archives often include a
// version-stamped top-level directory (golangci-lint-1.61.0-linux-amd64/...)
// that would otherwise need to be re-encoded per release.
func ExtractBinary(archivePath string, archiveType ArchiveType, wantBinary, destPath string) error {
	switch archiveType {
	case ArchiveRaw:
		return installRaw(archivePath, destPath)
	case ArchiveTarGz:
		return extractFromTar(archivePath, wantBinary, destPath, openGzip)
	case ArchiveTarXz:
		return extractFromTar(archivePath, wantBinary, destPath, openXz)
	case ArchiveZip:
		return extractFromZip(archivePath, wantBinary, destPath)
	default:
		return fmt.Errorf("unsupported archive type: %q", archiveType)
	}
}

func installRaw(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

type readerOpener func(io.Reader) (io.Reader, error)

func openGzip(r io.Reader) (io.Reader, error) {
	return gzip.NewReader(r)
}

func openXz(r io.Reader) (io.Reader, error) {
	return xz.NewReader(r)
}

func extractFromTar(archivePath, wantBinary, destPath string, open readerOpener) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	decompressed, err := open(f)
	if err != nil {
		return fmt.Errorf("decompress %s: %w", archivePath, err)
	}
	tr := tar.NewReader(decompressed)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if path.Base(hdr.Name) != wantBinary {
			continue
		}
		return writeStream(tr, destPath)
	}
	return fmt.Errorf("binary %q not found in archive %s", wantBinary, archivePath)
}

func extractFromZip(archivePath, wantBinary, destPath string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open zip %s: %w", archivePath, err)
	}
	defer func() { _ = zr.Close() }()
	for _, zf := range zr.File {
		if strings.HasSuffix(zf.Name, "/") {
			continue
		}
		if path.Base(zf.Name) != wantBinary {
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			return fmt.Errorf("open zip entry %s: %w", zf.Name, err)
		}
		err = writeStream(rc, destPath)
		_ = rc.Close()
		return err
	}
	return fmt.Errorf("binary %q not found in archive %s", wantBinary, archivePath)
}

func writeStream(r io.Reader, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, r); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
