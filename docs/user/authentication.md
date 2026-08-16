---
sidebar_position: 2
---

# Authentication & TLS

## Authentication

Omada controllers require every Open API call to include the Omada Cloud Controller
ID (`omadacId`) in the path and an access token in the `Authorization` header. Despite
the token response reporting `"tokenType": "bearer"`, the controller does **not**
accept the standard `Authorization: Bearer <token>` scheme — it silently rejects it
with a misleading `errorCode -44112` ("The access token has expired...") regardless
of how fresh the token is. The real scheme is `Authorization: AccessToken=<token>`,
which is what this SDK's `auth.AuthenticationProvider` sends
(see [`auth/authentication_provider.go`](https://github.com/NerdIT-Tech/tplink-omada-sdk-for-go/blob/main/auth/authentication_provider.go));
this was confirmed against a live controller, not documented in the OpenAPI
description itself. This SDK ships two ways to obtain that token — see
[`auth/`](https://github.com/NerdIT-Tech/tplink-omada-sdk-for-go/tree/main/auth):

**Client credentials** (recommended for applications): register a client ID/secret
under *Settings > Platform Integration* on the controller, and let the SDK fetch and
transparently re-issue the access token for you:

```go
client, err := sdk.NewWithClientCredentials(
    "https://192.168.1.1:8043", // controller base URL
    clientID, clientSecret,
    omadacID, // from GET {baseURL}/api/info, unauthenticated
    sdk.NewInsecureHTTPClient(), // self-signed cert on the local controller
)
```

**Application-managed bearer token**: if your application already owns an access
token (and its lifecycle), attach it directly:

```go
client, err := sdk.NewWithAccessToken(baseURL, accessToken, httpClient)
```

Every generated call is then made through the fluent builder, e.g.:

```go
page, pageSize := int32(1), int32(10)
resp, err := client.Openapi().V1().ByOmadacId(omadacID).Sites().Get(ctx, &openapi.V1ItemSitesRequestBuilderGetRequestConfiguration{
    QueryParameters: &openapi.V1ItemSitesRequestBuilderGetQueryParameters{
        Page:     &page,
        PageSize: &pageSize,
    },
})
```

## TLS

Pass `nil` as the `httpClient` argument for a default client with certificate
verification enabled, or `sdk.NewInsecureHTTPClient()` to skip verification for
controllers with self-signed certificates (the common case for local Omada
controllers). Pass your own `*http.Client` for full control (custom CA, proxies,
timeouts, etc.).
