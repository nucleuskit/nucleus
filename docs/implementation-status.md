# Implementation Status

This document records what the current checkout actually implements after the
agent-native redesign work. Nucleus is being narrowed into an AI-facing protocol
layer for Go services, not a project scaffold, provider SDK collection, or
platform control plane.

## Status Key

| Status | Meaning |
| --- | --- |
| Implemented | Code exists, has tests, and is expected to work in the current checkout. |
| Metadata-only | Produces local inspection or evidence metadata without wiring business handlers or provider SDKs. |
| Removed | The old scaffold/provider/platform behavior has been deleted from the CLI surface. |
| In progress | The redesign document calls for more work before this area is complete. |

## CLI Surface

| Command | Current status | Notes |
| --- | --- | --- |
| `nucleus adopt` | Implemented | Adds a minimal protocol index to an existing project. Writes only `nucleus.yaml` and `.nucleus/*`; does not generate business code or modify `go.mod/go.sum`. |
| `nucleus init` | Removed | The old service/worker/library template generator was deleted. Adoption is the entry path. |
| `nucleus capability add` | Removed | Provider scaffold, provider defaults, `postgres` special cases, and dependency writes were deleted from the CLI surface. Capability/provider choices must be recorded as decisions. |
| `nucleus describe` | Implemented | Emits structured project facts. A project without `nucleus.yaml` is still describable and receives a `manifest.missing` warning. |
| `nucleus mark` | Implemented | Declares `contract`, `capability`, and `verify` anchors in `nucleus.yaml` only. Capability symbols are written as stable resolved IDs when found, recorded as declared intent when missing, and rejected with candidates when ambiguous. |
| `nucleus trace` | Implemented | MVP supports `trace symbol` over `symbol_graph`, `trace route` over the conservative flow graph, and `trace capability` over manifest capability anchors. Ambiguous short symbol names fail with candidates instead of guessing. |
| `nucleus impact` | Implemented | MVP supports `impact symbol`, `impact file`, and `impact contract`; outputs affected symbols, files, tests, routes, and graph edges with confidence metadata. |
| `nucleus plan` | Implemented | Produces task-oriented edit surfaces, commands, risks, decision-only capability context, read-only recipe candidates, `blocked_decisions` for locked provider/library/driver conflicts, and best-effort `impact_summary` with affected symbols, files, routes, contracts, tests, capabilities, graph edges, and verification commands. It no longer emits provider hints or provider scaffold commands. |
| `nucleus decision validate` | Implemented | Validates `.nucleus/decisions` evidence against `decision.v1`, manifest capability declarations, edit surfaces, verification commands, locked decision hashes, and supersede hashes. It remains read-only. |
| `nucleus decision accept` | Implemented | Explicitly accepts one decision file by setting `status: accepted`, `locked: true`, `accepted_by`, `accepted_at`, and canonical `decision_hash`. Writes only the selected decision file. |
| `nucleus decision supersede` | Implemented | Explicitly fills `supersedes_hash` for one supersede decision from the referenced prior decision. Writes only the selected decision file and does not accept it. |
| `nucleus validate` | Implemented | Validates manifest and contract files when present. |
| `nucleus gen` | Implemented | Generates contract-derived artifacts. It should remain contract-only and must not create application wiring or provider glue. |
| `nucleus lint` | Implemented | Checks protocol and safety rules without binding capabilities to fixed provider/library imports. HTTP remains contract-backed. |
| `nucleus verify` | Implemented | Runs protocol validation, strict lint, decision quality validation, generated freshness, and only the verification commands declared in `nucleus.yaml`. Decision diagnostics are included in the top-level `evidence.v1` output. |
| `nucleus apply` | Implemented | Applies plan file edits only after edit-surface, path traversal, and symlink checks. It deliberately does not execute shell commands and emits `evidence.v1`. |
| `nucleus execute` | Implemented | Executes allowlisted plan commands and emits `evidence.v1`. |
| `nucleus repair` | Implemented | Supports bounded repair for generated freshness and single explicit patch candidates with expected hashes. It emits `evidence.v1` and is not a general code-repair engine. |
| `nucleus scenario` | Implemented | Builds OpenAPI/error-derived scenario suggestions and can run explicit HTTP scenario checks when a base URL or handler is provided. Runnable HTTP checks emit `evidence.v1`. |
| `nucleus serve` | Metadata-only | Serves `/healthz`, `/readyz`, and `/.well-known/nucleus.json` metadata. It can inspect projects without a manifest. |
| `nucleus report` | Metadata-only | Summarizes local AI task quality, decision quality, locked decision drift, and recipe candidate usage. Platform readiness, upload payloads, release dry-runs, and control-plane fields were removed. |
| `nucleus mcp --stdio` | Metadata-only | Serves local stdio MCP tools for agents. Tools expose service description, edit surfaces, contracts, capabilities, trace, impact, symbol lookup, decision validation, reports, plans, and read-only recipes. |
| `nucleus migrate` | Removed | Version migration and compatibility planning was removed because the project is pre-release and should not carry downgrade/compatibility paths. |

## Contract and Manifest Layer

| Area | Current status | Notes |
| --- | --- | --- |
| Manifest parsing | Implemented | Runtime structs now use manifest v2 objects for contracts and capabilities. Strict YAML parsing rejects removed `nucleus`, provider, library, driver, and platform fields instead of silently accepting them. |
| No-manifest inspection | Implemented | `describe` infers service name/version from `go.mod` or directory name and emits structured facts instead of failing. |
| Symbol graph | Implemented | `describe` emits `symbol_graph` with stable Go symbol IDs, package/file/type/interface/function/method nodes, calls, implements, test relation edges, and `source/confidence/stale` edge metadata. |
| OpenAPI inspection | Implemented | Loads route metadata, request parameters, request body shapes, examples, and validation hints from `api/openapi.yaml`. |
| Proto inspection | Implemented | Extracts package, service, method, streaming, and `google.api.http` facts with a lightweight parser. |
| Error catalog | Implemented | Validates stable error codes, messages, and HTTP status mappings from `api/errors.yaml`. |
| Generated freshness | Implemented | Compares contract source hashes with generated target markers declared in `ai.generated`. |

## JSON Result Contracts

Core CLI JSON outputs are being converged on `result_kind`, `schema_version`,
`schema_ref`, `ok`, and `diagnostics`. Result schemas are materialized for
adoption, marking, description, validation, linting, generation, scenarios,
planning, decisions, tracing, impact analysis, serving, reporting, MCP
structured results, recipe result views, and execution evidence. `plan` fails
`ok` when top-level diagnostics contain errors, not only when edit or decision
blockers are present.

## Capabilities And Providers

Capabilities are semantic anchors. They are not provider implementations.

Provider, library, driver, ORM, SDK, queue, database, and observability choices
belong in structured decision evidence under `.nucleus/decisions`, not in
`nucleus.yaml` and not in Go hard-coded defaults.

Capability kind suggestions are stored as data in
`cmd/nucleus/internal/capvocab/capability-kinds.v1.json`. That vocabulary is
advisory only: it contains names, aliases, descriptions, and planning flags, but
no providers, drivers, libraries, default choices, DSNs, or dependency metadata.

Recipe knowledge is loaded from project-local `.nucleus/recipes/*.yaml`,
`.yml`, or `.json` files plus a small built-in read-only recipe set. Project
recipes override built-ins with the same id. Recipes are strict, read-only
knowledge documents: unknown fields such as executable `commands` fail
validation, and recipe data is not allowed to write files, modify dependencies,
or become an accepted decision. `plan` exposes safe matches as
`context.recipe_candidates` with `source` and a `candidate_only` selection
policy. Recipe suggestions do not modify plan commands, generated outputs,
provider decisions, dependencies, or locked decision state.

## Runtime Modules

The runtime modules remain separate Go modules. They should be treated as
optional user/project dependencies, not as dependencies introduced by `adopt` or
capability commands.

| Module | Current status | Notes |
| --- | --- | --- |
| `github.com/nucleuskit/http` | Existing runtime module | HTTP server, route registration, response envelopes, middleware, clients, CORS, SSE, static assets, and well-known metadata exist outside the protocol CLI. |
| `github.com/nucleuskit/grpc` | Existing runtime module | gRPC server/client wrappers, discovery resolver support, health/reflection options, and interceptor chains exist outside the protocol CLI. |
| `github.com/nucleuskit/worker` | Existing runtime module | Worker primitives exist outside the protocol CLI. |

## Adoption Guidance

Use `nucleus adopt --dir . --json` inside an existing Go project to add the
minimal protocol index. Then use:

```bash
nucleus mark capability order_store --kind relational_store --symbol OrderStore --json
nucleus describe --dir . --json --flow
nucleus plan --dir . --task "<change>" --json
nucleus mcp --stdio
nucleus verify --dir . --json
```

Do not use Nucleus to generate a project layout, select providers, or wire
third-party SDKs. Those choices belong to the user project and must be captured
as explicit decision evidence.
