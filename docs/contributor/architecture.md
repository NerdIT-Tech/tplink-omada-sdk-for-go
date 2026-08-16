---
sidebar_position: 1
slug: /
---

# Architecture

This guide is for people modifying the SDK itself — regenerating it from a newer
Omada API description, fixing the hand-written auth/response layer, or maintaining
CI and releases. If you're building an application *against* the SDK, see the
[User Guide](/docs/) instead.

## Module layout

The repository is two separate Go modules:

- **`.`** (module `github.com/NerdIT-Tech/tplink-omada-sdk-for-go`) — the SDK itself.
- **`bdd/`** — a separate module containing the Godog BDD suite that exercises the
  SDK against a real, reachable controller (see
  [Testing against a live controller](testing.md)). It's split out so the SDK's own
  `go.mod` doesn't carry Godog and its dependencies as a requirement for every
  consumer.

Both modules are built, vetted, linted, and tested independently in CI — see
[CI & Release](ci-and-release.md).

## Generated vs. hand-written code

Almost everything in the repo is generated and should not be hand-edited:

- **`openapi/`** — one `*_request_builder.go` file per path (and path-parameter
  segment) in the OpenAPI description; ~2,100 files.
- **`models/`** — one file per OpenAPI schema (request/response bodies, enums,
  nested objects); ~2,700 types.
- **`omada_api_client.go`** — the root `OmadaApiClient` entry point.

All three are produced by a single `kiota generate` invocation — see
[Regenerating the SDK](regenerating-the-sdk.md) for the exact command and the one
post-generation fix that currently needs to be re-applied by hand every time.

The hand-written layer is small and deliberately kept that way:

- **`client.go`** — the two public constructors (`NewWithClientCredentials`,
  `NewWithAccessToken`), `NewInsecureHTTPClient`, and
  `omadaResponseNormalizingTransport`.
- **`auth/`** — `AuthenticationProvider` (attaches the Omada access token to every
  outgoing request) and the two `AccessTokenProvider` implementations,
  `StaticTokenProvider` and `ClientCredentialsTokenProvider`.

## Why `client.go` normalizes responses

`omadaResponseNormalizingTransport` (in `client.go`) exists because a live Omada
controller was observed to violate its own response contract in two ways that would
otherwise cause the generated Kiota adapter to silently lose information:

1. **Missing `Content-Type`.** Most responses — success or error — are labeled
   `application/json`, but at least one error path (an unrecognized `omadacId`) sends
   a well-formed JSON envelope with no `Content-Type` at all. Kiota picks its parser
   by `Content-Type`; with none, it returns `(nil, nil)` for the whole call, discarding
   the envelope entirely.
2. **Envelope wrapped in a non-2xx status.** Most controller-level errors (invalid
   `omadacId`, out-of-range page, expired token) arrive as the controller's
   `{"errorCode":...,"msg":...}` envelope wrapped in HTTP `200` — the generated
   builders deserialize that normally. But some failures (observed for a missing
   `Authorization` header) send that same envelope shape wrapped in a non-2xx status
   instead, and since no error mapping is registered for those statuses, the adapter
   discards the body and returns a generic "unexpected status code" error.

The transport reads the (typically small, paginated) response body once and: sets
`Content-Type: application/json` when it's missing but the body looks like JSON, and
rewrites the status to `200` when the body is a recognizable Omada envelope
(`isOmadaErrorEnvelope`). Both fixes make errors reach the caller through the normal
`OperationResponse` result — see [Using the SDK](/docs/usage) for how consumers are
expected to read that envelope. Regression coverage for both fixes lives in
`client_test.go`, using `httptest` servers that reproduce the exact malformed
responses observed from a live controller.

## Why `auth/` exists as a separate package

The controller reports `"tokenType": "bearer"` in its token response but does **not**
accept the standard `Authorization: Bearer <token>` header — it silently rejects it
with a misleading `errorCode -44112` regardless of token freshness. The real scheme is
`Authorization: AccessToken=<token>`, which `auth.AuthenticationProvider` sends
unconditionally. This was confirmed against a live controller and isn't documented in
the OpenAPI description itself, which is why it's centralized in one place rather than
left for each caller to discover independently. See
[Authentication & TLS](/docs/authentication) for the user-facing side of this.
