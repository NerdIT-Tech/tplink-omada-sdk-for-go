# Security Policy

## Supported Versions

This project follows [Semantic Versioning](https://semver.org/). Until a `v1.0.0`
release, only the latest published `0.x` release receives security fixes.

| Version        | Supported          |
| -------------- | ------------------- |
| Latest release | :white_check_mark:  |
| Older releases | :x:                 |

## Reporting a Vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Report it privately via [GitHub Security Advisories](https://github.com/NerdIT-Tech/tplink-omada-sdk-for-go/security/advisories/new)
for this repository. This opens a private discussion with the maintainer and lets
us coordinate a fix and disclosure timeline before any details become public.

Please include:

- A description of the vulnerability and its potential impact.
- Steps to reproduce, or a minimal proof-of-concept.
- The affected version(s) / commit.

We aim to acknowledge new reports within 5 business days, and to agree on a
disclosure timeline once the issue is confirmed.

## Scope Notes

- `openapi/` and `models/` are generated from `omada-open-api-sec.json` via
  [Kiota](https://github.com/microsoft/kiota) and are not hand-audited line by
  line; the request/response transport contract they share (`client.go`,
  `auth/`) is the layer this project actively hardens and tests — see
  [`QA_REPORT.md`](QA_REPORT.md).
- This SDK talks to a caller-supplied Omada controller over HTTPS. Vulnerabilities
  in the Omada Controller software itself are out of scope here — report those to
  TP-Link.
- Automated scanning: this repo runs [CodeQL](.github/workflows/codeql.yml),
  [gosec](.github/workflows/ci.yml), and [govulncheck](.github/workflows/ci.yml)
  on every change, plus a weekly [OSSF Scorecard](.github/workflows/scorecard.yml)
  supply-chain assessment.
