# Nucleus Workflow Reference

## Empty Repository

Use CLI bootstrap only:

```bash
nucleus init --name <service-name> --module <module-path> --template service --agent codex --dir .
```

Ask for only missing bootstrap inputs:

- service name
- Go module path
- template type: `service`, `worker`, or `library`

Do not inspect or copy local Nucleus examples, testdata, templates, or source checkouts to derive business code.

## Existing Service

Start with:

```bash
nucleus describe --dir . --json --pretty
nucleus plan --dir . --task "<task>" --json --executable
```

Use `describe` to determine current facts:

- service identity and tier
- endpoints and gRPC services
- error codes
- dependencies
- capability graph
- allowed, readonly, and forbidden edit surfaces
- generated freshness
- verification commands

Use `plan --executable` to determine the candidate write set. Treat `blocked_edits[]` as a hard stop unless the user explicitly changes the service edit boundaries. Do not continue by adding provider imports or wiring outside the manifest path.

## HTTP Or gRPC Behavior Change

Apply contract edits before implementation:

1. Update `api/openapi.yaml` for HTTP behavior.
2. Update `api/proto/*.proto` for gRPC behavior.
3. Update `api/errors.yaml` for new or changed errors.
4. Run `nucleus gen --dir .`.
5. Implement domain, adapter, or app wiring in allowed edit surfaces.
6. Run lint and verification.

Do not hand-write routes that drift from OpenAPI. Do not invent error codes outside `api/errors.yaml`.

## Capability Or Provider Change

Apply manifest edits before code:

1. Update `nucleus.yaml` capability declarations, dependencies, providers, or edit boundaries.
2. If MCP is available, call `get_capability_recipe` and use its `scaffold_command` or provider-specific `bridge_candidates[].scaffold_command`.
3. Prefer a scaffold command when available, for example `nucleus capability add sql --provider postgres --dir .`.
4. Add or adjust app wiring in the service assembly layer.
5. Keep `domain` independent of provider SDKs and transport frameworks.
6. Verify manifest/import consistency with `nucleus lint --dir . --strict`.

SQL providers such as PostgreSQL and MySQL belong to `cap/sql`. MongoDB is a separate document-store capability and must not be modeled as SQL.

## Fix Or Repair

Use evidence first:

```bash
nucleus verify --dir . --json > evidence.json
nucleus repair --dir . --from-evidence evidence.json --max-rounds 1
```

Repair is limited to allowed edit surfaces. If the failure requires readonly or forbidden files, missing business intent, secrets, or unclear rollback, report `needs_manual_action`.

## Handoff

Summarize:

- changed files
- contract and manifest changes
- generated outputs
- verification commands and results
- residual risks
- manual follow-ups
