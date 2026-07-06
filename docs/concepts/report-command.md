# Report Command

`nucleus report` produces local, auditable summaries for AI-assisted change
quality. It is intentionally read-only: the command does not upload artifacts,
call a control plane, or contact provider SDKs.

## Modes

The only mode is `ai_quality`. It reads task result JSON files from
`artifacts/nucleus/ai-tasks` under the service directory. Use `--ai-tasks` to
point at an explicit directory. A missing default directory is reported as a
warning and yields a zero-task report; a missing explicit directory is an error.

## Output

Human output is the default. `--json` emits a stable CLI result envelope:

```json
{
  "result_kind": "nucleus.report_result",
  "schema_version": "report.v1",
  "schema_ref": "contract/schema/report.v1.schema.json",
  "ok": true,
  "mode": "ai_quality",
  "summary": {},
  "diagnostics": []
}
```

The report schema lives at `contract/schema/report.v1.schema.json`. Quality
payloads are nested under `ai_quality`; shared rollups stay in `summary`.

## AI Quality Inputs

AI task result files may be simple task summaries with `plan_pass`,
`apply_pass`, and `verify_pass`, or evidence replay wrappers with
`kind: "nucleus.evidence_replay"` and an `evidence` payload. Evidence replay is
used to count real verification and repair outcomes without requiring live
network or provider dependencies.

Capability event summaries only count explicit event records. The command does
not infer provider behavior from secrets, environment variables, or hidden
global state.
