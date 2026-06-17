# Platform Mapping

This document records how `nucleus report --platform` maps local service facts to
release-readiness metadata. The report is intentionally local-only: it does not
upload artifacts, call a control plane, or require provider SDK credentials.

## Local Artifacts

`platform_upload_payload` describes the service metadata that a future platform
upload step may consume:

- service identity from `nucleus.yaml`
- declared capabilities
- HTTP endpoints and gRPC services from contract inspection
- generated freshness state
- verification commands

`release_dry_run` describes the local release matrix and verification commands
that should be checked before publishing a build.

## Readiness Gates

The report emits readiness gates for:

- local platform upload payload metadata
- local release dry-run metadata
- generated freshness
- declared verification commands

Failed gates do not prevent report generation. Release automation should inspect
the gates and decide whether to continue.

## Provider Strategy

Provider strategy is derived from the capability graph. Providers remain
optional bridge or application wiring choices; the report never imports provider
SDKs or turns detected providers into defaults.
