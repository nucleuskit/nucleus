# Nucleus

[![CI](https://github.com/nucleuskit/nucleus/actions/workflows/ci.yml/badge.svg)](https://github.com/nucleuskit/nucleus/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

Nucleus is an agent-native Go microservice protocol layer for structured service inspection, bounded AI edits, technical decision evidence, and local verification.

It is designed for AI agents, CI systems, and human reviewers to work from the same local facts: service contracts, source graphs, manifest indexes, generated freshness, safe edit surfaces, decision evidence, and verification evidence.

> Status: pre-alpha. The public repository is being prepared for the first source import under `github.com/nucleuskit/nucleus`.

For a code-level view of what is implemented, removed, and still in progress,
see [Implementation Status](docs/implementation-status.md).

For hands-on usage, see [Nucleus 使用指南](docs/usage.md).

## Why Nucleus

Traditional Go microservice frameworks are optimized for humans writing code inside an IDE. Nucleus is optimized for AI agents understanding and changing existing Go services safely and repeatably.

Nucleus focuses on four rules:

- **Contract-first**: OpenAPI, protobuf, and error definitions are the source of truth for external behavior.
- **Manifest as protocol index**: service identity, contracts, capability anchors, AI edit surfaces, and verification commands are declared in `nucleus.yaml`.
- **Graph-first inspection**: symbols, calls, interfaces, routes, tests, and generated freshness are exposed as machine-readable local facts.
- **AI-safe loop**: `adopt -> mark -> describe graph -> trace -> impact -> plan -> decision -> verify`, with optional `mcp --stdio` access for agents, makes service changes reviewable, reproducible, and bounded.

Nucleus is not a full-stack middleware bundle, project scaffold, provider SDK collection, or platform control plane. Redis, Kafka, SQL, Nacos, tracing exporters, ORMs, and similar infrastructure are user-project decisions recorded as explicit decision evidence.

## Core Concepts

### Contract

A Nucleus service treats these files as public behavior contracts:

- `api/openapi.yaml` for HTTP APIs
- `api/proto/*.proto` for gRPC APIs
- `api/errors.yaml` for stable error codes and HTTP mappings

Generated artifacts, when used, are contract-derived only. They must carry freshness metadata and remain outside normal business edit surfaces.

### Manifest

`nucleus.yaml` describes the service identity and machine-readable protocol index:

- service name and metadata
- declared contracts
- capability anchors and symbols
- edit surfaces for AI agents
- verification commands for CI and local review

Provider, library, ORM, and driver choices do not live in `nucleus.yaml`; they belong in `.nucleus/decisions/*.yaml`.

### Capability Anchors

Nucleus separates capability declaration from infrastructure implementation:

- capabilities are semantic anchors, not provider implementations
- capability kinds are advisory vocabulary, not hard-coded enums
- provider decisions live in `.nucleus/decisions/*.yaml`
- recipes may suggest approaches, but cannot write files or execute commands
- domain code and application wiring remain owned by the user project

### AI-Safe Change Loop

The intended workflow inside a service directory is:

```text
nucleus adopt --json
nucleus mark contract http --kind openapi --path api/openapi.yaml
nucleus mark capability order_store --kind relational_store --symbol OrderStore
nucleus describe --json
nucleus trace symbol <symbol> --json
nucleus trace capability order_store --json
nucleus impact symbol <symbol> --json
nucleus plan --task "change request" --json
nucleus decision validate .nucleus/decisions/<decision>.yaml
nucleus gen
nucleus lint
nucleus verify
nucleus report --json
nucleus mcp --stdio
```

The goal is not just to generate code. The goal is to produce evidence that the change respected contracts, manifests, source graphs, locked decisions, generated freshness, edit surfaces, and verification commands.

`adopt` adds a minimal protocol index to an existing Go project. It does not
create business directories, choose providers, or modify `go.mod`.

`describe` emits service facts that agents and reviewers consume. `trace` and
`impact` expose call chains and change blast radius, while `plan` embeds a
best-effort impact summary next to edit surfaces and verification commands.
`lint` checks protocol consistency, safety boundaries, decision evidence,
generated freshness, and stale graphs without requiring a fixed project layout.
`verify` executes only manifest-declared verification commands and emits
evidence.

`gen` writes reproducible contract-derived artifacts only. It must not create
application wiring, provider glue, runtime bootstraps, or dependency changes.
Use `nucleus gen --json` when automation needs auditable evidence; the result
includes `result_kind: "nucleus.gen_result"`, `ok`, `source_hash`, generated
`files`, `summary`, and validation `diagnostics`.

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
`schema_version: "serve-result.v1"`, `schema_ref:
"contract/schema/serve-result.v1.schema.json"`, `ok`, `mode`, `summary`,
`diagnostics`, and `server`.

`report` summarizes local quality without making network calls or referencing a
control plane. It reports graph coverage, decision quality, verification status,
edit-surface violations, generated freshness, AI task evidence, unresolved
symbols, and locked decision changes.

`mcp` exposes the same local facts as structured stdio MCP tools for agents.
It is read-only metadata access over contracts, edit surfaces, capabilities,
symbol search, trace, impact, decisions, report, plan, and local recipes.

Projects without OpenAPI, proto, or error catalogs are still valid Nucleus
projects. They run in graph-only mode until a task touches external behavior.

## Project Shape

The main repository hosts the CLI, protocol schemas, source inspection code, and public documentation:

```text
cmd/nucleus/         CLI implementation
contract/schema/     Protocol schemas
contract/inspect/    Source and contract inspection
docs/                Concepts, ADRs, and plans
```

The root repository keeps only the protocol boundary in-tree. Runtime,
provider, ORM, transport, and storage implementations are project decisions,
not framework defaults:

- `github.com/nucleuskit/contract`

Projects can still depend on any router, RPC library, worker runtime, database
driver, ORM, or provider SDK they choose. Nucleus only records the contract,
capability anchors, decision evidence, and generated protocol glue needed for
agents to understand and change the code safely.

## Roadmap

The early public roadmap is intentionally narrow:

- Publish the source under `github.com/nucleuskit/nucleus`.
- Stabilize the first `adopt`, `describe graph`, `trace`, `impact`, `plan`, `decision`, `verify`, and local MCP loop.
- Add protocol schemas for manifest v2, decisions, recipes, graphs, reports, diagnostics, and evidence.
- Remove scaffold, provider bridge, and platform-readiness paths from the core boundary.
- Keep generated artifacts contract-derived and readonly by default.
- Publish `v0.1.0-alpha.1` once the public module path, CI, examples, and docs are aligned.

## Contributing

Nucleus is early, but contributions should already follow the project boundaries:

- Start with contracts and manifests when changing external behavior.
- Keep protocol code small and dependency-aware.
- Prefer explicit capability anchors and decisions over hidden framework defaults.
- Add tests or verification evidence with behavior changes.
- Do not add provider SDK adapters, scaffold defaults, or compatibility shims for private or legacy internal SDKs.

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
