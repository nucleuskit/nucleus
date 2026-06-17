# Report Command

`nucleus report` produces local, auditable summaries for AI-assisted change
quality and release readiness. It is intentionally read-only: the command does
not upload artifacts, call a control plane, or contact provider SDKs.

## Modes

The default mode is `ai_quality`. It reads task result JSON files from
`artifacts/nucleus/ai-tasks` under the service directory. Use `--ai-tasks` to
point at an explicit directory. A missing default directory is reported as a
warning and yields a zero-task report; a missing explicit directory is an error.

`--platform` switches to `platform_readiness`. This mode reuses
`contract/inspect` service metadata, generated freshness, verification commands,
and capability graph facts. Platform upload and release dry-run fields are local
artifact metadata, not network actions.

`--platform` and `--ai-tasks` are mutually exclusive because they select
different input models.

## Output

Human output is the default. `--json` emits a stable CLI result envelope:

```json
{
  "result_kind": "nucleus.report_result",
  "schema_version": "report.v1",
  "schema_ref": "contract/schema/report.schema.json",
  "ok": true,
  "mode": "ai_quality",
  "summary": {},
  "diagnostics": []
}
```

The report schema lives at `contract/schema/report.schema.json`. Mode-specific
payloads are nested under `ai_quality` or `platform_readiness`; shared rollups
stay in `summary`.

## AI Quality Inputs

AI task result files may be simple task summaries with `plan_pass`,
`apply_pass`, and `verify_pass`, or evidence replay wrappers with
`kind: "nucleus.evidence_replay"` and an `evidence` payload. Evidence replay is
used to count real verification and repair outcomes without requiring live
network or provider dependencies.

Capability event summaries only count explicit event records. The command does
not infer provider behavior from secrets, environment variables, or hidden
global state.

## Platform Readiness

Platform readiness gates summarize whether local metadata is ready for a
subsequent release or upload step:

- `platform_upload_payload`: local payload metadata can be emitted.
- `release_dry_run`: local release matrix metadata can be emitted.
- `generated_freshness`: generated targets match their contract sources.
- `verification_commands`: service verification commands are declared.

Risk gates are advisory. They describe provider SDK scope, control-plane network
scope, and generated freshness risk. Failed gates do not prevent the report from
being emitted; consumers should inspect gate fields before release automation.
