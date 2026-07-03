# Nucleus Workflow Reference

## Empty Or Existing Repository

Use Nucleus adoption only after the user project exists:

```bash
nucleus adopt --dir . --agent codex
```

Ask for only missing adoption inputs:

- service name

If the directory has no Go module or business code yet, ask the user how they want to initialize their own project. Do not use Nucleus examples, testdata, templates, or source checkouts to derive business code.

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

Use `plan --executable` to determine the candidate write set. Treat `blocked_edits[]` as a hard stop unless the user explicitly changes the service edit boundaries. Do not continue by adding provider imports or wiring outside the allowed edit surface.

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

1. Update `nucleus.yaml` capability declarations, dependencies, or edit boundaries.
2. Record provider, library, ORM, driver, SDK, and wiring choices as structured decision evidence under `.nucleus/decisions`.
3. Add or adjust user-project interfaces and implementation wiring inside allowed edit surfaces.
4. Keep domain code independent of provider SDKs and transport frameworks unless the project already owns that boundary.
5. Verify manifest, contract, and import consistency with `nucleus lint --dir . --strict`.

Do not write provider, library, driver, or ORM decisions into `nucleus.yaml`. Those are decisions, not protocol index fields.

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
