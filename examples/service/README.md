# Nucleus Service Template

This directory documents the generated service template. It is not itself a
checked-in runnable generated service.

Generate a runnable service with:

```bash
nucleus init --template service --name demo-api --module example.com/demo-api --dir ./demo-api
```

The generated service includes:

- `api/openapi.yaml` and `api/errors.yaml` as contract SSOT.
- `contract/gen` for contract facts and freshness markers.
- `internal/adapter/http/gen` for generated HTTP route binders and handler interfaces.
- `cmd/<service>/main.go` as process entry only.
- `internal/app`, `internal/config`, `internal/domain`, `internal/usecase`, `internal/adapter`, `internal/component`, and `internal/server` as handwritten service surfaces.
- `configs/config.example.yaml`, `docs/`, `deploy/`, and `test/integration/` scaffolding.
- `AGENTS.md` with business-service edit rules and top-level directory registration.

From the generated service directory, run:

```bash
nucleus validate --dir .
nucleus lint --dir . --strict
nucleus verify --dir . --json
go test ./...
```

See [Implementation Status](../../docs/implementation-status.md) for the current
module and adapter status.
