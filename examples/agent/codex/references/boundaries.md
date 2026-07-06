# Nucleus Boundary Reference

## Product Boundary

Nucleus is an agent-native Go microservice protocol layer: contract facts, manifest indexes, decision evidence, graph inspection, edit surfaces, and safe AI editing workflow.

It is not a general application scaffold, UI wizard, business domain model, middleware bundle, provider SDK collection, or compatibility layer for unrelated frameworks.

## Layer Rules

- Project structure belongs to the user repository, not to Nucleus.
- Use the repository's own `AGENTS.md`, existing packages, and local conventions before creating paths.
- Keep domain code independent of provider SDKs and transport frameworks unless the project already owns that boundary.
- Keep provider/library/driver choices in project code and `.nucleus/decisions`, not in `nucleus.yaml`.
- Treat Nucleus runtime modules as optional project dependencies, not as dependencies introduced by Nucleus adoption.

## Contract-First

External behavior starts in contracts:

- HTTP: `api/openapi.yaml`
- gRPC: `api/proto/*.proto`
- errors: `api/errors.yaml`

Run generation after contract edits. Treat generated files as readonly unless a Nucleus command regenerated them.

## Manifest-First

Service identity, contracts, capabilities, dependencies, and AI edit boundaries start in `nucleus.yaml`.

Capability code should match manifest declarations. Adding provider code without manifest declaration or decision evidence is drift.

For business services, adding a capability means updating `nucleus.yaml` with a v2 capability object, recording provider/library/driver choices as `.nucleus/decisions` evidence, and implementing the user-project interface inside allowed edit surfaces.

Relational storage, document storage, cache, message bus, metrics, tracing, and logging are semantic capability kinds. PostgreSQL, MySQL, GORM, Xorm, Redis, Kafka, Zap, OpenTelemetry, and similar choices are user/project decisions, not Nucleus manifest fields.

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
