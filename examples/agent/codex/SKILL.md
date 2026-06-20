---
name: nucleus
description: Use when creating, inspecting, modifying, verifying, or repairing Go services built with Nucleus, the Contract/Manifest-first AI-native Go service kernel. Triggers include Nucleus business services, nucleus.yaml, api/openapi.yaml, api/errors.yaml, api/proto, capability declarations, edit surfaces, generated freshness, executable plans, and Nucleus CLI/MCP workflows.
---

# Nucleus

## Start Here

Treat Nucleus as an installed service-kernel tool. Do not derive business code by copying a local Nucleus source checkout, examples, or testdata.

For an empty directory, bootstrap through the CLI:

```bash
nucleus init --name <service-name> --module <module-path> --template service --agent codex --dir .
```

Use `--template worker` for worker services and `--template library` for library packages.

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
4. Apply Manifest-First edits for service identity, dependencies, capabilities, providers, and AI edit boundaries in `nucleus.yaml`.
5. Write only paths allowed by `describe.edit_surfaces.allowed`.
6. For an existing capability scaffold, prefer the CLI entrypoint:

   ```bash
   nucleus capability add <capability> --provider <provider> --dir .
   ```

   When MCP is available, call `get_capability_recipe` first and follow its `scaffold_command` / `bridge_candidates[].scaffold_command`. The CLI supports every registered Nucleus capability. Some providers generate deep wiring, while others create a manifest/provider placeholder that must be replaced with real bridge construction in `internal/app`. For PostgreSQL persistence, use `nucleus capability add sql --provider postgres --dir .`. `sql` is the capability; `postgres` is a provider/bridge choice. MongoDB is not SQL; use `mongo` capability flows when available.
7. Regenerate instead of manually editing generated files:

   ```bash
   nucleus gen --dir .
   ```

8. Verify with the service's declared commands, plus the Nucleus defaults when applicable:

   ```bash
   nucleus validate --dir .
   nucleus lint --dir . --strict
   nucleus verify --dir . --json
   go test ./...
   ```

9. Repair only from evidence and only inside allowed edit surfaces:

   ```bash
   nucleus repair --dir . --from-evidence evidence.json --max-rounds 1
   ```

10. Report changed files, commands run, evidence, residual risks, and any manual follow-up.

## Hard Stops

Stop and report the blocker instead of guessing when:

- `nucleus.yaml` is missing and required bootstrap inputs are unavailable.
- A requested edit falls outside `describe.edit_surfaces.allowed`.
- The plan includes `blocked_edits[]` that would be required for the task.
- A Manifest-First capability/provider change requires `nucleus.yaml` but it is blocked; do not add provider imports or app wiring as a workaround.
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
