# Migrate Command

`nucleus migrate` produces a local migration plan for moving a service between
Nucleus or manifest schema versions. It is intentionally report-only: the
command does not rewrite service files, call provider SDKs, or contact a control
plane.

## Modes

The default mode is `plan`. It loads the service through `contract/inspect`,
combines the discovered manifest, contract, capability, generated freshness, and
verification metadata with CLI-local migration rules, then emits an auditable
plan.

`--check` switches to readiness checking. The output shape remains the same, but
readiness failures such as stale generated artifacts or missing verification
commands become errors and produce a non-zero exit code.

Both modes require `--from-version` and `--to-version`. Versions use
`MAJOR.MINOR` or `MAJOR.MINOR.PATCH` format with an optional leading `v`.

## Output

Human output is the default. `--json` emits a stable CLI result envelope:

```json
{
  "result_kind": "nucleus.migrate_result",
  "schema_version": "migrate.v1",
  "schema_ref": "contract/schema/migrate.schema.json",
  "ok": true,
  "mode": "plan",
  "summary": {},
  "diagnostics": [],
  "migration": {}
}
```

Use `--pretty` with `--json` for indented JSON. Use `--report <path>` to write
the same JSON envelope to a report file. Report paths must resolve inside the
service directory; relative paths are resolved from that directory.

## Migration Scope

The migration plan is contract-first. Manifest and agent instructions are listed
before contract surfaces, generated artifacts, capability wiring, and
verification commands. Exact version rules are recorded in the CLI command; when
no exact rule is registered, the command emits a warning and falls back to a
generic forward migration checklist.

`migrate` does not replace the AI-safe change loop. Service owners should apply
the planned edits through:

```bash
nucleus describe --dir . --json
nucleus plan --dir . --task "<migration task>" --json
nucleus gen --dir .
nucleus lint --dir . --strict
nucleus verify --dir . --json
```

This keeps migration behavior auditable without expanding Nucleus into a
runtime framework or hidden auto-upgrader.
