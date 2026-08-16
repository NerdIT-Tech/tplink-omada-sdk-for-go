---
sidebar_position: 3
---

# Testing against a live controller

[`bdd/`](https://github.com/NerdIT-Tech/tplink-omada-sdk-for-go/tree/main/bdd) is a
separate Go module containing a [Godog](https://github.com/cucumber/godog)
(Gherkin) BDD suite that exercises this SDK against a real, reachable controller —
see the `.feature` files under `bdd/features` for the user stories covered
(authentication, TLS handling, listing sites, listing devices). See
[`QA_REPORT.md`](https://github.com/NerdIT-Tech/tplink-omada-sdk-for-go/blob/main/QA_REPORT.md)
for the bugs this suite found in the hand-written
`auth`/`client.go` layer (now fixed, with regression coverage in both the BDD suite
and `client_test.go`) and one documented, unfixed limitation.

Configure a `.env` file at the repo root (git-ignored):

```
CLIENT_ID="..."
CLIENT_SECRET="..."
BASE_URL="https://192.168.1.1:8043/"
TLS_VERIFY="false"
```

Then run:

```sh
cd bdd
go test ./steps/... -v
```
