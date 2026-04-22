package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Writer errors on stdout/stderr are non-actionable, so these helpers swallow
// them to keep call sites readable. Used across the cli package.
func fpf(w io.Writer, format string, args ...any) { _, _ = fmt.Fprintf(w, format, args...) }
func fpln(w io.Writer, args ...any)               { _, _ = fmt.Fprintln(w, args...) }

// resolveRepoRoot returns the effective repo root: --repo flag, env, or cwd.
func resolveRepoRoot() string {
	if flags.repoRoot != "" {
		p, _ := filepath.Abs(flags.repoRoot)
		return p
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
