<!-- PR title must be a Conventional Commit line, e.g. `fix(auth): fail fast on empty access token` — CI lints it, and release-please uses it to compute the next version and CHANGELOG entry. -->

## What & why

<!-- Short description of the change and the problem it solves. Link related issues. -->

## Checklist

- [ ] `gofmt -s -w .`, `golangci-lint run ./...`, and `go test ./...` pass locally (root module)
- [ ] `bdd/` still builds/vets (`cd bdd && go build ./... && go vet ./...`) if touched — live scenarios need a reachable controller and aren't run in CI
- [ ] New or changed exported surface has unit tests
- [ ] **Generated code:** if `openapi/`, `models/`, or `omada-open-api-sec.json` changed via `kiota generate`, the `GetWanList`/`Serialize` fix documented in the README's "Regenerating the SDK" section was re-applied
- [ ] `CHANGELOG.md` is untouched (release-please manages it)

<!-- If this PR intentionally skips tests or docs, say why: -->
