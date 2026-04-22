# Changelog

All notable changes to Aegis are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-04-22

First public release. Implements the v1.0 scope of the project specification.

### Added

#### CLI + pipeline

- Commands: `init`, `install`, `uninstall`, `run` (with `--hook` and `--check`),
  `doctor`, `baseline`, `ignore`, `fmt`, `explain`, `version`.
- Nine-stage execution pipeline: config load → validate → staged-file detect →
  stack detect → binary resolve + SHA256 verify → parallel run → normalize →
  filter (allowlist / baseline / inline / warn_paths) → gate → report.
- Scanner adapters: `gitleaks`, `opengrep`, `osv-scanner`, `biome`, `ruff`,
  `golangci-lint`, `gofmt`, `shellcheck`.
- Stack auto-detect: `npm`, `python`, `go`, `shell`.
- Git hook integration for `pre-commit` and `pre-push`, with detection and
  delegation to pre-existing foreign hooks (Husky, lefthook, etc.).

#### Output

- Pretty terminal output with color, icons, and a deterministic sort order.
- JSON output with a stable, documented schema. Includes `checks_run` so
  consumers can distinguish "ran and clean" from "filtered out".
- Exit codes per spec §11.5: `0` ok, `1` blocking, `2` config, `3` binary
  resolve, `4` scanner crash.

#### Supply-chain model

- SHA256 pin per platform for every scanner binary; verified on every run.
- `strict_versions: true` by default - refuses to execute unverified binaries.
- `protect_secrets: true` by default - the secrets check cannot be disabled
  or bypassed (`AEGIS_SKIP=secrets`, inline ignores, and `--no-verify` all
  still run it).
- Override mechanism (`AEGIS_SKIP` + mandatory `AEGIS_REASON`) with an
  append-only audit log at `.aegis/overrides.log`.
- Release artifacts cross-compiled with `CGO_ENABLED=0` and signed with
  Sigstore keyless.

#### Project infrastructure

- MIT License.
- Community health files: `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`,
  `SECURITY.md`, `CHANGELOG.md`, `CODEOWNERS`.
- Issue and pull-request templates; Dependabot config for Go modules,
  GitHub Actions, and pip.
- Reference `aegis.yaml` configs under `examples/` for go-service,
  typescript-monorepo, and python-lib.
- MkDocs-Material documentation site at
  https://mhchlagou.github.io/aegis/, deployed via native GitHub Pages.

#### CI / release workflows

All third-party actions SHA-pinned; least-privilege `permissions:` scoped
per job.

- `ci.yml`: cross-platform test (ubuntu / macos / windows), lint
  (golangci-lint v2.11.4), matrix build (5 platforms), govulncheck,
  end-to-end self-smoke.
- `codeql.yml`: Go static analysis, weekly cron + on every PR.
- `pr-checks.yml`: Conventional Commits title validation + size label.
- `docs.yml`: build and deploy docs via `actions/deploy-pages`.
- `stale.yml`: automated cleanup of inactive issues/PRs.
- `release.yml`: cross-compile, sign with Sigstore, publish GitHub Release
  on `v*` tag push.

### Requirements

- Go **1.25+** for building from source.
- External scanner binaries per your `aegis.yaml`. Aegis coordinates them
  but does not bundle or download them - install and pin each one you use.

[Unreleased]: https://github.com/MHChlagou/aegis/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/MHChlagou/aegis/releases/tag/v0.1.0
