package installer

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// allowedHosts is the permissive set: GitHub release infrastructure plus
// the vendor domains that ship our current scanners. Every entry is trust
// surface — a compromise of the host's release artifacts would let an
// attacker ship bytes that still pass our sha256 check (they'd own both
// the file and, eventually, the pin generated from it). Extend only when
// adding a scanner that cannot be sourced from a host already on the list.
var allowedHosts = map[string]bool{
	"github.com":                    true, // releases for gitleaks, opengrep, osv-scanner, biome, ruff, golangci-lint, shellcheck
	"objects.githubusercontent.com": true, // GitHub release asset CDN (302 target)
	"astral.sh":                     true, // ruff nightlies / future channels
	"biomejs.dev":                   true, // biome vendor CDN
}

// checkHostAllowlistWith checks against the given allowlist. Separated from
// the default-allowlist variant so callers (tests, private-mirror builds)
// can inject their own set without mutating package state.
func checkHostAllowlistWith(rawURL string, hosts map[string]bool) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse url %q: %w", rawURL, err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("refusing non-HTTPS download: %s", rawURL)
	}
	host := strings.ToLower(u.Hostname())
	if !hosts[host] {
		return fmt.Errorf("host %q is not on the allowlist (allowed: %v); refusing to download %s", host, sortedHosts(hosts), rawURL)
	}
	return nil
}

func sortedHosts(hosts map[string]bool) []string {
	out := make([]string, 0, len(hosts))
	for h := range hosts {
		out = append(out, h)
	}
	sort.Strings(out)
	return out
}
