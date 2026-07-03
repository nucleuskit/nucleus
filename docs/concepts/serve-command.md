# Serve Command

`nucleus serve` runs a local metadata-only HTTP surface for a service directory.
It is intentionally narrow: the command loads manifest and contract facts through
`contract/inspect`, then exposes health, readiness, and well-known Nucleus
metadata endpoints. It does not wire provider SDKs, start generated business
handlers, or replace an application's own runtime assembly.

## Modes

The default mode is `serve`. It inspects the service, prints a startup result,
then listens on `--addr`:

```bash
nucleus serve --dir . --addr 127.0.0.1:8080
```

`--check` switches to inspection-only mode. It produces the same result envelope
without opening a listener, which makes it suitable for CI and agent preflight
checks:

```bash
nucleus serve --dir . --check --json
```

## Endpoints

The local metadata server exposes:

- `GET /healthz`: liveness probe returning `OK`
- `GET /readyz`: compact JSON readiness summary
- `GET /.well-known/nucleus.json`: full `contract/inspect` service description

Non-GET requests to these metadata endpoints return `405 Method Not Allowed`.

## Output

Human output is the default. `--json` emits a stable CLI result envelope:

```json
{
  "result_kind": "nucleus.serve_result",
  "schema_version": "serve-result.v1",
  "schema_ref": "contract/schema/serve-result.v1.schema.json",
  "ok": true,
  "mode": "check",
  "summary": {
    "status": "ready",
    "service": "demo",
    "version": "0.1.0",
    "addr": "127.0.0.1:8080",
    "served_paths": ["/healthz", "/readyz", "/.well-known/nucleus.json"],
    "endpoint_count": 1,
    "grpc_service_count": 0,
    "capability_count": 1,
    "generated_fresh": false,
    "errors": 0,
    "warnings": 0
  },
  "diagnostics": [],
  "server": {
    "listening": false,
    "network_scope": "loopback",
    "metadata_endpoints": [
      {
        "method": "GET",
        "path": "/healthz",
        "content_type": "text/plain"
      },
      {
        "method": "GET",
        "path": "/readyz",
        "content_type": "application/json"
      },
      {
        "method": "GET",
        "path": "/.well-known/nucleus.json",
        "content_type": "application/json"
      }
    ]
  }
}
```

Use `--pretty` with `--json` for indented JSON. Inspection failures are reported
as `serve.inspect_failed` diagnostics and return a non-zero exit code. By
default, `serve` refuses non-loopback bind addresses such as `0.0.0.0:8080`;
use `--allow-non-local` only when exposing local metadata beyond loopback is
intentional.

## Scope

`serve` is a local metadata command, not a framework host. Business routes should
remain owned by generated adapters and application wiring. Scenario execution
stays explicit through `nucleus scenario --run-http --base-url ...`, so Nucleus
continues to provide auditable evidence without hidden runtime auto-wiring.
