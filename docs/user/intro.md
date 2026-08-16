---
sidebar_position: 1
slug: /
---

# Introduction

`tplink-omada-sdk-for-go` is a Go SDK for the TP-Link Omada Controller Open API,
generated with [Kiota](https://github.com/microsoft/kiota) from the controller's own
OpenAPI description (`omada-open-api-sec.json`). It covers the full API surface
(1,500+ endpoints, 2,700+ schemas) as strongly-typed, fluent request builders, plus a
small hand-written layer (`client.go`, `auth/`) that handles the controller's
non-standard authentication scheme and a couple of response-format inconsistencies —
see [Using the SDK](usage.md) for how those pieces fit together.

## Install

```sh
go get github.com/NerdIT-Tech/tplink-omada-sdk-for-go
```

## Quickstart

```go
import sdk "github.com/NerdIT-Tech/tplink-omada-sdk-for-go"

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

## Where to go next

- **[Authentication & TLS](authentication.md)** — the two ways to construct a client,
  and an important gotcha with the controller's `Authorization` header scheme.
- **[Using the SDK](usage.md)** — the fluent request builder pattern, reading
  responses, error handling, and a worked example. Read this once; it applies
  uniformly to every endpoint the controller exposes.
- **[pkg.go.dev API reference](https://pkg.go.dev/github.com/NerdIT-Tech/tplink-omada-sdk-for-go)**
  — full generated API documentation for every request builder and model.

If you're looking to modify the SDK itself — regenerate it from a newer API
description, run the BDD suite against a live controller, or understand the CI/release
setup — see the [Contributor Guide](/contributing/) instead.
