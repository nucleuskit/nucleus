# Nucleus

[![CI](https://github.com/nucleuskit/nucleus/actions/workflows/ci.yml/badge.svg)](https://github.com/nucleuskit/nucleus/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

Nucleus is an AI-first Go microservice kernel for contract-driven service generation, evolution, and verification.

It is designed for AI agents, CI systems, and human reviewers to work from the same source of truth: service contracts, manifests, generated freshness, safe edit surfaces, and verification evidence.

> Status: pre-alpha. The public repository is being prepared for the first source import under `github.com/nucleuskit/nucleus`.

For a code-level view of what is implemented, what is scaffold-only, and the
current module release status, see [Implementation Status](docs/implementation-status.md).

## Why Nucleus

Traditional Go microservice frameworks are optimized for humans writing code inside an IDE. Nucleus is optimized for AI agents changing services safely and repeatably.

Nucleus focuses on four rules:

- **Contract-first**: OpenAPI, protobuf, and error definitions are the source of truth for external behavior.
- **Manifest-first**: service identity, capabilities, dependencies, and AI edit surfaces are declared in `nucleus.yaml`.
- **Thin kernel**: core lifecycle, identity, errors, context, and response types stay small and dependency-aware.
- **AI-safe loop**: `describe -> plan -> lint -> verify` makes service changes reviewable, reproducible, and bounded.

Nucleus is not a full-stack middleware bundle. Redis, Kafka, SQL, Nacos, tracing exporters, and similar infrastructure are connected through explicit capability interfaces and optional bridges.

## Core Concepts

### Contract

A Nucleus service treats these files as public behavior contracts:

- `api/openapi.yaml` for HTTP APIs
- `api/proto/*.proto` for gRPC APIs
- `api/errors.yaml` for stable error codes and HTTP mappings

Generated handlers, types, clients, and freshness metadata are derived from these contracts.

### Manifest

`nucleus.yaml` describes the service identity and machine-readable operating surface:

- service name, version, tier, and ownership
- declared capabilities and providers
- dependencies and contract snapshots
- edit surfaces for AI agents
- verification commands for CI and local review

### Capability Protocol

Nucleus separates capability declaration from infrastructure implementation:

- `github.com/nucleuskit/cap/*` defines small interfaces, options, and no-op implementations.
- `github.com/nucleuskit/bridge/*` provides optional adapters.
- Application wiring injects bridges into runtime code.
- Domain code stays independent from transport and infrastructure SDKs.

### AI-Safe Change Loop

The intended workflow inside a service directory is:

```text
nucleus describe --json
nucleus validate
nucleus plan --task "change request" --json
nucleus gen
nucleus scenario --json
nucleus serve --check --json
nucleus lint
nucleus verify
nucleus report --json
```

The goal is not just to generate code. The goal is to produce evidence that the change respected contracts, manifests, generated freshness, dependency boundaries, and verification commands.

`describe` emits the service facts that agents and reviewers consume. `validate`
checks manifest and contract source legality before later workflow steps. `lint`
checks project conventions and risk rules, while `verify` executes the validation,
lint, build, and test evidence pipeline.

`gen` writes reproducible artifacts under generated targets such as
`contract/gen` and `internal/adapter/http/gen`. Use `nucleus gen --json` when
automation needs auditable evidence; the result includes
`result_kind: "nucleus.gen_result"`, `ok`, `source_hash`, generated `files`,
`summary`, and validation `diagnostics`.

`scenario` turns OpenAPI routes, error catalogs, and flow inspection into
reviewable test suggestions. Use `nucleus scenario --json` for a scenario plan,
`nucleus scenario --draft-cases --json` to emit executable HTTP case drafts, and
`nucleus scenario --run-http --base-url http://127.0.0.1:8080 --json` to produce
explicit opt-in HTTP scenario evidence. Network execution is never implicit:
`--run-http` and `--cases` require `--base-url`, and captured headers are
redacted for sensitive keys such as authorization and cookies.

`serve` exposes local metadata endpoints for manual probes and automation
preflight checks. Use `nucleus serve --check --json` to inspect the service
without opening a listener, or `nucleus serve --addr 127.0.0.1:8080` to expose
`/healthz`, `/readyz`, and `/.well-known/nucleus.json`. The command is
metadata-only: it does not auto-wire provider SDKs or generated business
handlers. JSON output uses `result_kind: "nucleus.serve_result"`,
`schema_version: "serve.v1"`, `schema_ref:
"contract/schema/serve.schema.json"`, `ok`, `mode`, `summary`, `diagnostics`,
and `server`.

`report` summarizes AI change quality and platform readiness without making
network calls. By default it reads AI task result JSON files from
`artifacts/nucleus/ai-tasks`; pass `--ai-tasks` for an explicit directory, or
`--platform` to emit local release-readiness metadata from `contract/inspect`.
Like other subcommands, the default output is human-readable and `--json` emits
a stable envelope with `result_kind: "nucleus.report_result"`,
`schema_version: "report.v1"`, `schema_ref:
"contract/schema/report.schema.json"`, `ok`, `mode`, `summary`, and
`diagnostics`.

For any service directory containing `nucleus.yaml` and contract sources, run
`nucleus validate --dir <service>`. Successful human output includes a short
validation summary; `--json` emits the same result with stable `ok`, `summary`,
and `diagnostics` fields.

## Project Shape

The main repository hosts the CLI, examples, and public documentation:

```text
api/                 Contract files
cmd/nucleus/         CLI implementation
examples/            Runnable examples and templates
docs/                Concepts, ADRs, plans, and platform mapping
```

The kernel, contract, capability, bridge, and runtime packages are published as
separate Go modules:

- `github.com/nucleuskit/core`
- `github.com/nucleuskit/contract`
- `github.com/nucleuskit/cap`
- `github.com/nucleuskit/bridge`
- `github.com/nucleuskit/http`
- `github.com/nucleuskit/grpc`
- `github.com/nucleuskit/worker`

## Roadmap

The early public roadmap is intentionally narrow:

- Publish the source under `github.com/nucleuskit/nucleus`.
- Stabilize the first `nucleus validate`, `gen`, `describe`, `plan`, `lint`, and `verify` loop.
- Keep `core` standard-library-only.
- Make `examples/hello-http` the first runnable contract-first service.
- Add focused capability protocols before adding optional bridges.
- Publish `v0.1.0-alpha.1` once the public module path, CI, examples, and docs are aligned.

## Contributing

Nucleus is early, but contributions should already follow the project boundaries:

- Start with contracts and manifests when changing external behavior.
- Keep kernel code small and dependency-aware.
- Prefer explicit capabilities over hidden framework defaults.
- Add tests or verification evidence with behavior changes.
- Do not add compatibility shims for private or legacy internal SDKs.

See [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md).

## License

Nucleus is licensed under the [Apache License 2.0](LICENSE).

## Star History

<a href="https://www.star-history.com/?repos=nucleuskit%2Fnucleus&type=date&legend=top-left">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=nucleuskit/nucleus&type=date&theme=dark&legend=top-left" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=nucleuskit/nucleus&type=date&legend=top-left" />
   <img alt="Star History Chart" src="https://api.star-history.com/chart?repos=nucleuskit/nucleus&type=date&legend=top-left" />
 </picture>
</a>
