# Nucleus Boundary Reference

## Product Boundary

Nucleus is an AI-native Go service kernel: Contract/Manifest SSOT, thin service kernel, capability protocol, runtime adapters, and safe AI editing workflow.

It is not a general application scaffold, UI wizard, business domain model, middleware bundle, or compatibility layer for unrelated frameworks.

## Layer Rules

- `core/*`: standard library only.
- `cap/*`: capability interfaces, options, and noop implementations.
- `bridge/*`: optional provider implementations of capabilities.
- `runtime/*`: transport assembly over `core` and `cap`; do not import `bridge`.
- business `domain/*`: depend on domain ports and core concepts only; do not import transport frameworks or provider bridges.
- business `internal/app`: assemble runtime, capabilities, providers, and bridges.

## Contract-First

External behavior starts in contracts:

- HTTP: `api/openapi.yaml`
- gRPC: `api/proto/*.proto`
- errors: `api/errors.yaml`

Run generation after contract edits. Treat generated files as readonly unless a Nucleus command regenerated them.

## Manifest-First

Service identity, capabilities, dependencies, providers, and AI edit boundaries start in `nucleus.yaml`.

Capability imports and app wiring must match manifest declarations. Adding provider code without manifest declaration is drift.

For business services, adding an existing capability means updating `nucleus.yaml` first, adding safe placeholder config, wiring providers only in `internal/app`, and verifying L004 with `nucleus lint --dir . --strict`. Use `nucleus capability add <capability> --provider <provider> --dir .` when the CLI has a scaffold.

`sql` is the relational database capability. PostgreSQL, generic `database/sql`, and GORM are provider/bridge choices for `cap/sql`; MongoDB is a separate document-store capability and should use `cap/mongo`.

## Edit Surfaces

Use `nucleus describe --dir . --json --pretty` before writing.

- `allowed`: maximum write surface.
- `readonly`: generated or reference output; update through generation.
- `forbidden`: never write.

If the task cannot be completed inside allowed surfaces, stop and ask for an explicit boundary change.

If a Manifest-First change requires `nucleus.yaml` and that file is blocked, stop at `blocked_edits[]`. Do not add imports, provider wiring, or repository code as a workaround for an undeclared capability.

## Directory Rules

Business service top-level directories come from that repository's `AGENTS.md`. Register a new top-level directory there before creating it.

Do not apply Nucleus kernel `.codex/steering/*.md` as business-service structure rules.

## Secrets

Do not add secrets, plaintext DSNs, tokens, local credentials, or environment-specific private values. Use placeholders and documentation.
