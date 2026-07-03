---
name: nucleus
description: Use when adopting, inspecting, modifying, verifying, or repairing Go services that use Nucleus as an agent-native protocol layer. Triggers include nucleus.yaml, .nucleus/decisions, api/openapi.yaml, api/errors.yaml, api/proto, capability declarations, edit surfaces, generated freshness, executable plans, and Nucleus CLI/MCP workflows.
---

# Nucleus

## Start Here

Treat Nucleus as an installed protocol tool. Do not derive business code by copying a local Nucleus source checkout, examples, or testdata.

For an existing Go project that has not adopted Nucleus yet, add only the local protocol index:

```bash
nucleus adopt --dir . --agent codex
```

If the repository is empty or has no Go module, ask the user how they want to initialize their own project first. Nucleus does not create service, worker, or library templates.

For an existing service, inspect machine-readable facts before editing:

```bash
nucleus describe --dir . --json --pretty
nucleus plan --dir . --task "<task>" --json --executable
```

Read `edit_surfaces`, `generated_freshness`, `capability_graph`, `verification.commands`, `edits[]`, and `blocked_edits[]`.

## Core Workflow

1. Inspect the service with `nucleus describe --dir . --json --pretty`.
2. Plan the requested change with `nucleus plan --dir . --task "<task>" --json --executable`.
3. Apply Contract-First edits for external behavior:
   - HTTP starts in `api/openapi.yaml`.
   - gRPC starts in `api/proto/*.proto`.
   - Error behavior starts in `api/errors.yaml`.
4. Apply Manifest-First edits for service identity, dependencies, capability indexes, and AI edit boundaries in `nucleus.yaml`.
5. Write only paths allowed by `describe.edit_surfaces.allowed`.
6. For a capability or provider decision, keep provider, library, ORM, driver, SDK, DSN, and wiring choices out of `nucleus.yaml`. Record the technical choice as structured decision evidence under `.nucleus/decisions`, then implement the user-project interface in paths allowed by the project.
7. Regenerate instead of manually editing generated files:

   ```bash
   nucleus gen --dir .
   ```

8. Verify with the service's declared commands:

   ```bash
   nucleus validate --dir .
   nucleus lint --dir . --strict
   nucleus verify --dir . --json
   ```

9. Repair only from evidence and only inside allowed edit surfaces:

   ```bash
   nucleus repair --dir . --from-evidence evidence.json --max-rounds 1
   ```

10. Report changed files, commands run, evidence, residual risks, and any manual follow-up.

## Hard Stops

Stop and report the blocker instead of guessing when:

- `nucleus.yaml` is missing and `nucleus adopt` cannot be run safely.
- A requested edit falls outside `describe.edit_surfaces.allowed`.
- The plan includes `blocked_edits[]` that would be required for the task.
- A Manifest-First capability change requires `nucleus.yaml` but it is blocked; do not add provider imports or app wiring as a workaround.
- HTTP, gRPC, or error behavior is requested without contract edits.
- Capability code is added without a matching `nucleus.yaml` declaration.
- A generated file would need manual editing.
- Verification commands are unknown, unavailable, or failing for unrelated reasons.
- Nucleus MCP output disagrees with CLI/schema output.

## Resource Map

Load only the reference needed for the task:

- `references/workflow.md`: end-to-end workflows for empty repos, existing services, feature changes, fixes, and handoff.
- `references/boundaries.md`: layer rules, capability protocol, edit surfaces, directory rules, and hard-stop details.
- `references/commands.md`: Nucleus CLI command selection and expected usage.
- `references/mcp.md`: how to use Nucleus MCP as a tool layer without bypassing CLI/schema facts.

Optional helper:

```bash
python3 <skill-dir>/scripts/nucleus_preflight.py --dir . --task "<task>" --out .tmp/nucleus-preflight
```

The helper only runs Nucleus CLI inspection/planning commands and writes local evidence files.
