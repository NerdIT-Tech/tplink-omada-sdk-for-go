---
sidebar_position: 2
---

# Regenerating the SDK

The client is generated with the Kiota CLI from `omada-open-api-sec.json`, the
controller's own OpenAPI description. That file isn't committed to the repo — fetch a
fresh copy from a reachable controller before regenerating:

```sh
curl -o omada-open-api-sec.json https://{controller-url}/v3/api-docs
```

Then:

```sh
kiota generate \
  -d omada-open-api-sec.json \
  -o . \
  -l Go \
  -c OmadaApiClient \
  -n github.com/NerdIT-Tech/tplink-omada-sdk-for-go \
  -m "application/json" -m "*/*"
```

The `-m "*/*"` structured MIME type is required: every response in this API
description is declared with content type `*/*`, and without it Kiota would treat
responses as opaque byte streams instead of deserializing them into models.

After regenerating, re-apply the fix in `models/check_wan_lan_status_open_api_v_o.go`
(`GetWanList`/`Serialize`): the spec's `wanList` items reference a schema
(`#/components/schemas/Lan infos`) that doesn't exist due to a naming inconsistency
in the description, so Kiota falls back to `UntypedNodeable` and emits an invalid
cast (`Parsable(&temp)` on an already-interface value) that doesn't compile; replace
it with `Parsable(v)`.
