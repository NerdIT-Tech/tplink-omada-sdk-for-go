---
sidebar_position: 4
---

# CI & Release

All workflows live under `.github/workflows/`. They share a few conventions worth
matching if you add or edit one: a `Harden Runner` step first in every job,
SHA-pinned `uses:` (with a `# vX.Y.Z` comment for humans), an explicit `permissions:`
block scoped to only what the job needs (the workflow-level default is
`permissions: {}`), and `workflow_dispatch` plus a path filter on workflows that don't
need to run on every PR.

## Pull request gates

- **`ci.yml`** — the main Go gate, PR-only by design (main is protected, so every
  commit on main already passed this against its PR head). Path-filtered to
  `**.go`, `go.mod`/`go.sum`, `bdd/go.mod`/`bdd/go.sum`, `.golangci.yml`. Runs, for
  both the root module and `bdd/`: `go mod download`/`verify`/`tidy` (fails on drift),
  `go build ./...` and `go vet ./...` across Ubuntu/macOS/Windows on the two most
  recent Go releases, `golangci-lint` (only-new-issues on the PR diff — pre-existing
  findings across the ~5k generated files under `openapi/`/`models/` are excluded via
  `.golangci.yml` since that code isn't hand-maintained), `go test` with coverage
  uploaded to Codecov, and `govulncheck`/`gosec` (SARIF uploaded to the Security tab).
- **`dependency-review.yml`** — path-filtered to the four `go.mod`/`go.sum` files;
  fails the PR on a newly introduced dependency with a high-severity advisory.
- **`pr-title.yml`** — enforces
  [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) on the PR
  title (`fix`, `feat`, `chore`, `docs`, `style`, `refactor`, `perf`, `test`, `ci`),
  since release-please (below) derives the changelog and version bump from commit
  type.
- **`lint-workflows.yml`** — lints every `**.yml`/`**.yaml` file in the repo with
  `actionlint` (which also picks up `shellcheck` for embedded `run:` blocks) and
  `yamllint -c .yamllint.yml`. Any new workflow file gets this for free.
- **`labeler.yml`** — applies path-based PR labels from `.github/labeler.yml` (e.g.
  `area: auth`, `area: openapi (generated)`, `type: documentation`).

## Security scanning (scheduled + push to main)

- **`codeql.yml`** — CodeQL static analysis for Go, on push/PR to `main` and weekly.
- **`scorecard.yml`** — [OSSF Scorecard](https://securityscorecards.dev/), on push to
  `main`, weekly, and on branch-protection-rule changes; publishes results and
  uploads SARIF.

## Automation

- **`dependabot-merge.yml`** — auto-merges Dependabot PRs for minor/patch updates
  only (major bumps are left for manual review).
- **`release.yml`** — runs `release-please-action` on every push to `main`, driven by
  `release-please-config.json` (Go release type, conventional-commit changelog
  sections, `bump-minor-pre-major: true` since the package is pre-1.0) and
  `.release-please-manifest.json` (current version). It opens/updates a release PR
  that bumps the version and changelog; merging that PR is what cuts an actual
  release/tag.

## Adding the docs site to this picture

`docs.yml` (added alongside this doc) follows the same conventions as the workflows
above but is scoped to `docs/**` and `website/**` only — it doesn't touch the Go
module and isn't gated by `ci.yml`. See
[Documentation](documentation.md) for what it builds and deploys.
