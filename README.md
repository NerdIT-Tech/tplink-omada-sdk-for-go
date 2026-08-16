# tplink-omada-sdk-for-go

[![CI](https://github.com/NerdIT-Tech/tplink-omada-sdk-for-go/actions/workflows/ci.yml/badge.svg)](https://github.com/NerdIT-Tech/tplink-omada-sdk-for-go/actions/workflows/ci.yml)
[![CodeQL](https://github.com/NerdIT-Tech/tplink-omada-sdk-for-go/actions/workflows/codeql.yml/badge.svg)](https://github.com/NerdIT-Tech/tplink-omada-sdk-for-go/actions/workflows/codeql.yml)
[![codecov](https://codecov.io/gh/NerdIT-Tech/tplink-omada-sdk-for-go/branch/main/graph/badge.svg)](https://codecov.io/gh/NerdIT-Tech/tplink-omada-sdk-for-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/NerdIT-Tech/tplink-omada-sdk-for-go.svg)](https://pkg.go.dev/github.com/NerdIT-Tech/tplink-omada-sdk-for-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/NerdIT-Tech/tplink-omada-sdk-for-go)](https://goreportcard.com/report/github.com/NerdIT-Tech/tplink-omada-sdk-for-go)

Go SDK for the TP-Link Omada Controller Open API, generated with [Kiota](https://github.com/microsoft/kiota)
from `omada-open-api-sec.json`. It covers the full API surface (1,500+ endpoints /
2,700+ schemas) as strongly-typed, fluent request builders under [`openapi/`](openapi)
and [`models/`](models).

## Install

```sh
go get github.com/NerdIT-Tech/tplink-omada-sdk-for-go
```

## Quickstart

```go
client, err := sdk.NewWithClientCredentials(
    "https://192.168.1.1:8043", // controller base URL
    clientID, clientSecret,
    omadacID, // from GET {baseURL}/api/info, unauthenticated
    sdk.NewInsecureHTTPClient(), // self-signed cert on the local controller
)

page, pageSize := int32(1), int32(10)
resp, err := client.Openapi().V1().ByOmadacId(omadacID).Sites().Get(ctx, &openapi.V1ItemSitesRequestBuilderGetRequestConfiguration{
    QueryParameters: &openapi.V1ItemSitesRequestBuilderGetQueryParameters{
        Page:     &page,
        PageSize: &pageSize,
    },
})
```

## Documentation

Full docs site: **https://NerdIT-Tech.github.io/tplink-omada-sdk-for-go/** (built from
the markdown below via [Docusaurus](website); see
[docs/contributor/documentation.md](docs/contributor/documentation.md) for how it's
set up).

**User Guide** — for applications built against the SDK:

- [Introduction](docs/user/intro.md) — install and a quickstart
- [Authentication & TLS](docs/user/authentication.md) — client construction options and an important gotcha with the controller's `Authorization` header scheme
- [Using the SDK](docs/user/usage.md) — the fluent request builder pattern, query parameters, reading responses, error handling, and a worked example

**Contributor Guide** — for working on the SDK itself:

- [Architecture](docs/contributor/architecture.md) — module layout and the generated-vs-hand-written boundary
- [Regenerating the SDK](docs/contributor/regenerating-the-sdk.md) — the Kiota command and a post-generation fix to re-apply
- [Testing against a live controller](docs/contributor/testing.md) — the `bdd/` Godog suite
- [CI & Release](docs/contributor/ci-and-release.md) — what each GitHub Actions workflow gates, and how release-please drives versioning
- [Documentation](docs/contributor/documentation.md) — how this doc split and the `website/` Docusaurus site work
- [QA Report](QA_REPORT.md) — bugs found and fixed during BDD testing
