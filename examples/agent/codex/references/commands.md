# Nucleus Command Reference

## Inspect

```bash
nucleus describe --dir . --json --pretty
```

Use this before editing. Read edit surfaces, generated freshness, capability graph, endpoints, errors, dependencies, and verification commands.

## Plan

```bash
nucleus plan --dir . --task "<task>" --json --executable
```

Use this to identify candidate edits, blocked edits, generated outputs, risk, and command order.

## Generate

```bash
nucleus gen --dir .
```

Run after contract changes. Do not manually edit generated files that should be produced by this command.

## Capability Scaffold

```bash
nucleus capability add <capability> --provider <provider> --dir .
```

Use this before manual provider wiring when adding a declared Nucleus capability. If MCP is available, first call `get_capability_recipe` and use its `scaffold_command` or provider-specific `bridge_candidates[].scaffold_command`.

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

Use this as the main machine-readable verification evidence. Also run any stricter commands listed under `describe.verification.commands`.

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
nucleus report --dir . --platform --json
```

Use when the task needs platform readiness metadata, quality metrics, or a structured handoff summary.
