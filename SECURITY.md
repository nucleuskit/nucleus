# Security Policy

## Supported Versions

Nucleus is pre-alpha. Until the first stable release, security fixes target the default branch and the latest published pre-release.

| Version | Supported |
| ------- | --------- |
| `main` | Yes |
| `v0.x` pre-releases | Best effort |

## Reporting a Vulnerability

Do not open a public issue for a vulnerability.

Use GitHub's private vulnerability reporting feature when available:

https://github.com/nucleuskit/nucleus/security/advisories/new

If private vulnerability reporting is not available, open a GitHub Security Advisory draft or contact the maintainers through the repository owner profile.

Please include:

- affected version or commit
- reproduction steps
- impact
- known mitigations
- whether the report can be publicly credited

## Scope

Security-sensitive areas include:

- generated service code and generated freshness checks
- contract parsing and schema validation
- `nucleus apply`, `execute`, `repair`, and other AI change automation
- edit surface enforcement
- manifest validation
- runtime request handling and response envelopes
- capability and bridge wiring
- CI, release, and provenance metadata

## Disclosure Process

Maintainers will acknowledge valid reports as soon as practical, assess severity, prepare a fix, and coordinate disclosure timing with the reporter when appropriate.
