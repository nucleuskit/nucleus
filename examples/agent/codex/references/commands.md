# Nucleus Command Reference

## Inspect

```bash
nucleus describe --dir . --json --pretty
```

Use this before editing. Read edit surfaces, generated freshness, capability graph, endpoints, errors, dependencies, and verification commands.
Also read `symbol_graph` when available; its node IDs are the stable IDs used by trace, impact, and mark workflows.

## Trace

```bash
nucleus trace symbol <symbol-or-id> --json
nucleus trace capability order_store --json
nucleus trace route "POST /orders" --json
```

Use this to inspect callers, callees, capability anchors, or route flow before editing. If a short symbol name is ambiguous, rerun with one of the returned stable symbol IDs.

## Impact

```bash
nucleus impact symbol <symbol-or-id> --json
nucleus impact file internal/order/store.go --json
nucleus impact contract api/openapi.yaml --json
```

Use this before planning or reviewing a change. Read affected symbols, files, tests, routes, and edge confidence; do not treat inferred edges as certain facts.

## Plan

```bash
nucleus plan --dir . --task "<task>" --json --executable
```

Use this to identify candidate edits, blocked edits, blocked locked decisions, generated outputs, risk, command order, and `impact_summary`. Read `blocked_decisions` before editing provider/library/driver code; if present, create and validate a supersede decision first. Read `impact_summary` before editing; empty or warning-only summaries mean you should run `trace` or `impact` explicitly. Treat `recipe_candidates` as source-labeled hints only; built-in candidates are read-only knowledge, not default provider choices.

## Adopt

```bash
nucleus adopt --dir . --agent codex
```

Use this to add a minimal protocol index to an existing Go project. It writes only `nucleus.yaml` and `.nucleus/*`.

## Mark

```bash
nucleus mark contract http --kind openapi --path api/openapi.yaml --json
nucleus mark capability order_store --kind relational_store --symbol OrderStore --json
nucleus mark verify "go test ./..." --json
```

Use this to declare manifest anchors. `mark` writes only `nucleus.yaml`; it does not generate implementation code, choose providers, or modify `go.mod/go.sum`.

## Generate

```bash
nucleus gen --dir .
```

Run after contract changes. Do not manually edit generated files that should be produced by this command.

## Capability Decisions

```bash
nucleus mark capability order_store --kind relational_store --symbol OrderStore --json
nucleus decision validate .nucleus/decisions/<decision>.yaml --json
```

Use `mark capability` for semantic anchors and `.nucleus/decisions` for provider/library/driver choices. Nucleus does not scaffold provider wiring or modify `go.mod/go.sum` for capabilities.

## Decision Validate

```bash
nucleus decision validate .nucleus/decisions/<decision>.yaml --json
nucleus decision accept .nucleus/decisions/<decision>.yaml --by human --json
nucleus decision supersede .nucleus/decisions/<decision-v2>.yaml --json
```

Use `validate` after creating or changing decision evidence. It checks manifest capability references, decision hashes, supersedes hashes, impact edit surfaces, and verification commands.

Use `accept` only after explicit human approval. It writes `status: accepted`, `locked: true`, `accepted_by`, `accepted_at`, and `decision_hash` into that one decision file.

Use `supersede` before replacing a locked provider/library/driver choice. It fills `supersedes_hash` from the referenced prior decision; it does not accept the new decision.

## Validate And Lint

```bash
nucleus validate --dir .
nucleus lint --dir . --strict
```

Use `validate` for YAML/OpenAPI shape and `lint` for Nucleus rules such as contract drift, layer boundaries, manifest/capability consistency, and generated freshness.

## Verify

```bash
nucleus verify --dir . --json
```

Use this as the main machine-readable verification evidence. It runs protocol checks and the commands declared under `nucleus.yaml` `verify.commands`; add missing project-owned checks with `nucleus mark verify "<command>"`.
Read the `decision` step before changing provider/library/driver code; failed decision diagnostics mean you need to fix or supersede decision evidence first.

## Execute

```bash
nucleus execute --dir . --plan plan.json --allow-command "<command>"
```

Use only when an executable plan explicitly includes controlled commands and the command is allowlisted.

## Repair

```bash
nucleus repair --dir . --from-evidence evidence.json --max-rounds 1
```

Use only from verification evidence. Do not repair through forbidden or readonly surfaces.

## Report

```bash
nucleus report --dir . --json
```

Use when the task needs local AI task quality metrics or a structured handoff summary. Do not treat it as platform readiness evidence.
Read `ai_quality.decision_quality` for locked decision drift and `ai_quality.recipe_candidate_usage` to see whether prior plans surfaced recipe candidates.
