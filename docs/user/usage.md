---
sidebar_position: 3
---

# Using the SDK

This guide covers the parts of the SDK that don't change from endpoint to endpoint:
how the client is put together, how the fluent request builders are shaped, how to
read a response, and how to handle errors. The API surface itself (1,500+ endpoints,
2,700+ schemas under
[`openapi/`](https://github.com/NerdIT-Tech/tplink-omada-sdk-for-go/tree/main/openapi)
and
[`models/`](https://github.com/NerdIT-Tech/tplink-omada-sdk-for-go/tree/main/models))
is generated directly from `omada-open-api-sec.json` by
[Kiota](https://github.com/microsoft/kiota), so once you know the pattern below it
applies uniformly to every resource the controller exposes — sites, devices, clients,
WLANs, VPN, hotspot/voucher management, and so on.

For authentication specifics, see [Authentication & TLS](authentication.md). For
regenerating the client, see
[Regenerating the SDK](/contributing/regenerating-the-sdk) in the Contributor Guide.
For running the BDD suite against a live controller, see
[Testing against a live controller](/contributing/testing), also in the Contributor
Guide.

## Installation

```sh
go get github.com/NerdIT-Tech/tplink-omada-sdk-for-go
```

The root package is `tplinkomadasdkforgo`; import it under the alias `sdk`, as the
examples in this repo do:

```go
import sdk "github.com/NerdIT-Tech/tplink-omada-sdk-for-go"
```

## Constructing a client

Two constructors build an `*sdk.OmadaApiClient`:

```go
client, err := sdk.NewWithClientCredentials(baseURL, clientID, clientSecret, omadacID, httpClient)
client, err := sdk.NewWithAccessToken(baseURL, accessToken, httpClient)
```

- `baseURL` is the controller's Open API base URL, e.g. `"https://192.168.1.1:8043"`.
- `httpClient` may be `nil` for a default, certificate-verifying client, or
  `sdk.NewInsecureHTTPClient()` for a self-signed local controller. You can also pass
  your own `*http.Client` for custom TLS, proxies, or timeouts.
- Both constructors wrap whatever `*http.Client` you pass with a response-normalizing
  transport (see [Error handling](#error-handling) below) — you don't need to do
  anything for that to take effect.

See [`authentication.md`](authentication.md) for the tradeoffs between the two and the
non-standard `Authorization` header scheme the controller requires.

## The fluent request builder pattern

Every controller endpoint is reachable as a chain of method calls starting from the
client, mirroring the URL path. Path segments become methods; path *parameters*
(the `{omadacId}`, `{siteId}`, etc. in the OpenAPI paths) become `ByXxx(id)` methods
that take the ID as an argument. The chain ends with an HTTP-verb method —
`Get`, `Post`, `Patch`, `Put`, or `Delete` — that actually issues the request.

```go
// GET /openapi/v1/{omadacId}/sites/{siteId}/devices
client.Openapi().V1().ByOmadacId(omadacID).Sites().BySiteId(siteID).Devices().Get(ctx, config)
```

Reading that chain against the path: `Openapi()` → `/openapi`, `V1()` → `/v1`,
`ByOmadacId(omadacID)` → `/{omadacId}`, `Sites()` → `/sites`,
`BySiteId(siteID)` → `/{siteId}`, `Devices()` → `/devices`. This mapping is completely
mechanical, so if you know the REST path from the
[Omada Open API documentation](https://www.tp-link.com/en/support/download/omada-software-controller/)
you can find its builder by walking the path segment by segment through your editor's
autocomplete on `client.Openapi().V1()...` — no need to search the generated source by
hand. Nested/related operations hang off the same builder as sibling methods, e.g.
`Sites()` also exposes `.Copy()`, `.Maintenance()`, `.MultiImport()`, `.Statistic()`,
`.Tags()`, and `.Template()` for `/sites/copy`, `/sites/maintenance`, etc.

Every verb method has the same shape:

```go
func (m *XxxRequestBuilder) Get(
    ctx context.Context,
    requestConfiguration *XxxRequestBuilderGetRequestConfiguration,
) (models.SomeResponseable, error)
```

`requestConfiguration` may be `nil` if the endpoint takes no query parameters, headers,
or options. Where it exists, it wraps:

- `QueryParameters` — a struct specific to that endpoint, e.g. `Page`, `PageSize`,
  `SearchKey`, `SortsName` for a listing endpoint. All fields are pointers; only set
  the ones you need. **Watch for endpoints where the controller treats a
  QueryParameters field as effectively required even though the Go type makes it
  optional** — e.g. `page`/`pageSize` on the devices listing endpoint. Omitting them
  compiles fine but the controller responds with a plain "Bad Request" instead of its
  usual structured error envelope (a documented, unfixed limitation — see
  `bdd/features/devices.feature`'s `@known-limitation` scenario).
- `Headers` / `Options` — request-level overrides; rarely needed for normal use.

Endpoints that take a body (`Post`, `Patch`, `Put`) take it as an explicit typed
parameter before `requestConfiguration`, e.g.
`Sites().Post(ctx, createSiteEntity, config)`.

## Query parameters and pagination

Listing endpoints follow a consistent shape: `Page` and `PageSize` (both `*int32`,
page numbers start at 1), often `SearchKey` for a fuzzy match, and `SortsXxx` fields
(`*string`, `"asc"`/`"desc"`) per sortable column.

```go
page, pageSize := int32(1), int32(50)
resp, err := client.Openapi().V1().ByOmadacId(omadacID).Sites().BySiteId(siteID).Devices().
    Get(ctx, &openapi.V1ItemSitesItemDevicesRequestBuilderGetRequestConfiguration{
        QueryParameters: &openapi.V1ItemSitesItemDevicesRequestBuilderGetQueryParameters{
            Page:     &page,
            PageSize: &pageSize,
        },
    })
```

The response's `Result` carries pagination metadata alongside the page of data — see
[Reading a response](#reading-a-response). A simple loop to walk every page:

```go
for page := int32(1); ; page++ {
    resp, err := client.Openapi().V1().ByOmadacId(omadacID).Sites().Get(ctx,
        &openapi.V1ItemSitesRequestBuilderGetRequestConfiguration{
            QueryParameters: &openapi.V1ItemSitesRequestBuilderGetQueryParameters{
                Page: &page, PageSize: &pageSize,
            },
        })
    if err != nil {
        return err
    }
    if resp.GetErrorCode() != nil && *resp.GetErrorCode() != 0 {
        return fmt.Errorf("controller error %d: %s", *resp.GetErrorCode(), *resp.GetMsg())
    }
    data := resp.GetResult().GetData()
    if len(data) == 0 {
        break // no more pages
    }
    for _, site := range data {
        // use site
    }
}
```

## Reading a response

Every operation returns two things: a generated response object and a Go `error`. The
`error` return is a **transport-level** failure only — DNS, connection refused, TLS,
context cancellation, or a non-2xx status the SDK couldn't map to a known envelope. A
`nil` error does **not** mean the call succeeded at the application level.

The Omada Open API wraps every response — success or failure — in the same envelope:

```json
{"errorCode": 0, "msg": "Success.", "result": { /* endpoint-specific payload, or absent */ }}
```

Generated response types (named `OperationResponseXxxable`, e.g.
`OperationResponseGridVOSiteSummaryInfoable`) mirror this exactly with three getters:

- `GetErrorCode() *int32` — `0` (or the envelope's absence, e.g. `Post` calls with no
  body describing a result use a generic `OperationResponseable`) means success;
  non-zero is a controller-side error.
- `GetMsg() *string` — human-readable message, always populated on error.
- `GetResult() Xxxable` — the typed payload. Endpoint-specific; for listing endpoints
  this is a `GridVOXxxable` with `GetData() []Xxxable`, `GetCurrentPage() *int32`,
  `GetCurrentSize() *int32`, and `GetTotalRows() *int64`.

**Always check both**, in this order:

```go
resp, err := client.Openapi().V1().ByOmadacId(omadacID).Sites().Get(ctx, config)
if err != nil {
    return fmt.Errorf("listing sites: %w", err)  // transport failure
}
if resp == nil {
    return errors.New("no response")             // see note below
}
if code := resp.GetErrorCode(); code == nil || *code != 0 {
    msg := ""
    if resp.GetMsg() != nil {
        msg = *resp.GetMsg()
    }
    return fmt.Errorf("controller rejected the request: %s", msg)
}
sites := resp.GetResult().GetData() // safe: errorCode == 0 implies a result
```

All model fields are pointers (Kiota's convention for distinguishing "absent" from
"zero value" in JSON). **Nil-check before dereferencing** — `GetMsg()`, `GetSiteId()`,
etc. can all be `nil`, particularly on error paths or optional fields. `GetResult()`
and `GetData()` return `nil`/an empty slice rather than panicking when there's nothing
there, so `len(resp.GetResult().GetData())` is safe even on an empty page — but only
call `GetResult()` on a response you've already confirmed has `errorCode == 0`,
since some non-success envelopes don't carry a `result` at all.

## Error handling

This SDK's `client.go` fixes two inconsistencies observed in the live controller's
error responses, so that **every controller-side error reaches you through the normal
`resp.GetErrorCode()`/`resp.GetMsg()` envelope** described above, rather than as a
missing result or an opaque transport error:

1. Some error responses omit `Content-Type` entirely, which would otherwise make the
   generated adapter silently return `(nil, nil)`.
2. Some error responses (observed for a missing `Authorization` header) wrap the usual
   `{errorCode, msg}` envelope in a non-2xx HTTP status instead of the `200` every
   other controller error uses, which would otherwise surface as a generic
   "unexpected status code" `error` with the real reason discarded.

Practical implications for your error handling:

- Prefer checking `resp.GetErrorCode()` over inspecting `err`'s text — the envelope is
  the controller's structured, documented error channel; `err` is reserved for cases
  the envelope mechanism can't represent (network failures, or the one documented
  limitation where a required-but-technically-optional query parameter is omitted and
  the controller returns a bare, envelope-free "Bad Request").
- An empty access token fails **locally**, before any request is sent, as
  `auth.ErrEmptyToken` — check with `errors.Is(err, auth.ErrEmptyToken)` if you need to
  distinguish "never authenticated" from a real transport failure.
- Sending the standard `Authorization: Bearer <token>` scheme is not a mistake you'll
  make directly (the SDK's `auth.AuthenticationProvider` always sends
  `Authorization: AccessToken=<token>`), but if you ever see `errorCode -44112`
  ("access token has expired") on a token you know is fresh, that misleading message is
  the controller's response to the wrong header scheme — see
  [Authentication & TLS](authentication.md).

## Building request bodies

`Post`/`Patch`/`Put` bodies are generated structs constructed with a `NewXxx()`
constructor and populated via `SetXxx(*T)` setters (mirroring the `GetXxx()` getters on
response models — pointers throughout, so wrap literals with `sdk` helpers like
`&value` or a local variable):

```go
site := models.NewCreateSiteEntity()
name := "Branch Office 12"
timezone := "America/New_York"
site.SetName(&name)
site.SetTimeZone(&timezone)

resp, err := client.Openapi().V1().ByOmadacId(omadacID).Sites().Post(ctx, site, nil)
```

Required vs. optional fields aren't enforced by the Go type system (everything is a
pointer); consult the field's doc comment in the generated model
(`models/create_site_entity.go` in this example) or the Omada Open API reference for
which fields the controller actually requires.

## Worked example: list sites, then that site's devices

```go
package main

import (
	"context"
	"fmt"
	"log"

	sdk "github.com/NerdIT-Tech/tplink-omada-sdk-for-go"
	"github.com/NerdIT-Tech/tplink-omada-sdk-for-go/openapi"
)

func main() {
	ctx := context.Background()

	client, err := sdk.NewWithClientCredentials(
		"https://192.168.1.1:8043",
		"<clientID>", "<clientSecret>", "<omadacID>",
		sdk.NewInsecureHTTPClient(),
	)
	if err != nil {
		log.Fatal(err)
	}

	page, pageSize := int32(1), int32(10)
	sitesResp, err := client.Openapi().V1().ByOmadacId("<omadacID>").Sites().
		Get(ctx, &openapi.V1ItemSitesRequestBuilderGetRequestConfiguration{
			QueryParameters: &openapi.V1ItemSitesRequestBuilderGetQueryParameters{
				Page: &page, PageSize: &pageSize,
			},
		})
	if err != nil {
		log.Fatalf("listing sites: %v", err)
	}
	if code := sitesResp.GetErrorCode(); code == nil || *code != 0 {
		log.Fatalf("controller rejected the request: %v", sitesResp.GetMsg())
	}
	sites := sitesResp.GetResult().GetData()
	if len(sites) == 0 {
		log.Fatal("no sites managed by this controller")
	}
	siteID := *sites[0].GetSiteId()

	devicesResp, err := client.Openapi().V1().ByOmadacId("<omadacID>").
		Sites().BySiteId(siteID).Devices().
		Get(ctx, &openapi.V1ItemSitesItemDevicesRequestBuilderGetRequestConfiguration{
			QueryParameters: &openapi.V1ItemSitesItemDevicesRequestBuilderGetQueryParameters{
				Page: &page, PageSize: &pageSize,
			},
		})
	if err != nil {
		log.Fatalf("listing devices: %v", err)
	}
	if code := devicesResp.GetErrorCode(); code == nil || *code != 0 {
		log.Fatalf("controller rejected the request: %v", devicesResp.GetMsg())
	}
	for _, d := range devicesResp.GetResult().GetData() {
		name, mac, status := "", "", int32(0)
		if d.GetName() != nil {
			name = *d.GetName()
		}
		if d.GetMac() != nil {
			mac = *d.GetMac()
		}
		if d.GetStatus() != nil {
			status = *d.GetStatus()
		}
		fmt.Printf("%s (%s) status=%d\n", name, mac, status)
	}
}
```

(Device `status` is an enum: `0` disconnected, `1` connected, `2` pending, `3`
heartbeat missed, `4` isolated — see the doc comment on `DeviceInfo.GetStatus` in
[`models/device_info.go`](https://github.com/NerdIT-Tech/tplink-omada-sdk-for-go/blob/main/models/device_info.go).)

## Finding the builder and model for an endpoint you need

With 2,118 generated files under `openapi/` there's no index worth hand-maintaining;
instead, navigate the same way the fluent chain does:

1. Start typing `client.Openapi().V1().ByOmadacId(omadacID).` in an editor with Go
   completion and let autocomplete list the available top-level resources
   (`Sites()`, `Devices()` action groups under `Devices`/`Clients`, etc.).
2. Chain further; each method name matches its OpenAPI path segment (camelCased). Verb
   methods (`Get`/`Post`/...) appear once you're at a leaf resource.
3. If you know the REST path already (from the Omada Open API reference PDF or
   `omada-open-api-sec.json`), its builder file is named by concatenating the path
   segments, e.g. `/openapi/v1/{omadacId}/sites/{siteId}/devices` →
   `openapi/v1_item_sites_item_devices_request_builder.go` (`item` marks each
   `{...}` path parameter).
4. Request/response model types live in `models/`, named after the OpenAPI schema
   (snake_cased filename, e.g. `SiteSummaryInfo` → `models/site_summary_info.go`).
   Every model type `Xxx` has a matching `Xxxable` interface — that's what request
   builder signatures use.
5. `go doc` works against generated code same as hand-written: e.g.
   `go doc ./openapi V1ItemSitesRequestBuilder` or
   `go doc ./models SiteSummaryInfo`.

## Testing your integration

See [Testing against a live controller](/contributing/testing) for running the
`bdd/` Godog suite against a live controller, and `client_test.go` for unit tests
covering the response-normalization transport and auth edge cases in isolation (using
`httptest`, no live controller needed) — both are good references for exercising your
own code against this SDK without a real Omada deployment on hand.
