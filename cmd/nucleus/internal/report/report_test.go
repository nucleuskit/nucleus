package report

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAIQualityReportCountsScenarioAndRealEvidenceSources(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "scenario.json", `{
  "id": "scenario-success",
  "source": "scenario",
  "labels": ["single_service"],
  "plan_pass": true,
  "apply_pass": true,
  "verify_pass": true,
  "repair_pass": false,
  "failure_located": false,
  "manual_action": false,
  "rollback_performed": false
}`)
	writeFile(t, dir, "real-evidence.json", `{
  "id": "missing-generated-real-evidence",
  "source": "real_evidence",
  "kind": "nucleus.evidence_replay",
  "labels": ["single_service", "repairable"],
  "evidence": {
    "kind": "nucleus.repair_evidence",
    "status": "repaired",
    "verification_pass": true,
    "rounds": [
      {
        "id": "repair-1",
        "strategy": "regenerate_missing_generated",
        "status": "repaired"
      }
    ]
  }
}`)

	report := mustAIQualityReport(t, dir)
	if report.ScenarioTaskCount != 1 {
		t.Fatalf("unexpected scenario_task_count: %#v", report)
	}
	if report.RealEvidenceTaskCount != 1 {
		t.Fatalf("unexpected real_evidence_task_count: %#v", report)
	}
	if report.SourceCoverageRate != 1.0 {
		t.Fatalf("expected both sources to be covered: %#v", report)
	}
	if report.RepairSuccessCount != 1 {
		t.Fatalf("expected replayed repair evidence to count as repair success: %#v", report)
	}
}

func TestAIQualityReportDoesNotInferReplaySuccessWithoutEvidence(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "ambiguous-real-evidence.json", `{
  "id": "ambiguous-real-evidence",
  "source": "real_evidence",
  "kind": "nucleus.evidence_replay",
  "labels": ["single_service"],
  "evidence": {
    "kind": "nucleus.repair_evidence",
    "status": "needs_manual_action"
  }
}`)

	report := mustAIQualityReport(t, dir)
	if report.FailureLocatedCount != 0 {
		t.Fatalf("ambiguous replay should not count as located failure: %#v", report)
	}
	if report.RepairableTaskCount != 0 || report.RepairSuccessCount != 0 {
		t.Fatalf("ambiguous replay should not become repairable or repaired: %#v", report)
	}
	if report.StrategySummary.ByTaskType["unspecified"].SuccessCount != 0 {
		t.Fatalf("ambiguous replay should not count as successful strategy summary: %#v", report.StrategySummary)
	}
}

func TestAIQualityReportOnlyReplaysExplicitEvidenceWrappers(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "scenario-with-evidence.json", `{
  "id": "scenario-with-evidence",
  "source": "scenario",
  "labels": ["single_service"],
  "plan_pass": true,
  "apply_pass": true,
  "verify_pass": true,
  "evidence": {
    "kind": "nucleus.verify_result",
    "ok": false
  }
}`)

	report := mustAIQualityReport(t, dir)
	if report.ScenarioTaskCount != 1 || report.RealEvidenceTaskCount != 0 {
		t.Fatalf("plain scenario task with evidence should not be replayed: %#v", report)
	}
	if report.FirstPassCount != 1 || report.FailedTaskCount != 0 {
		t.Fatalf("plain scenario task pass flags should be preserved: %#v", report)
	}
}

func TestAIQualityReportTreatsVerifyStepPassFalseAsLocatedFailure(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "verify-replay.json", `{
  "id": "verify-replay",
  "kind": "nucleus.evidence_replay",
  "labels": ["single_service"],
  "evidence": {
    "result_kind": "nucleus.verify_result",
    "ok": false,
    "steps": [
      {
        "id": "generated_freshness",
        "phase": "generated_freshness",
        "pass": false
      }
    ]
  }
}`)

	report := mustAIQualityReport(t, dir)
	if report.FailureLocatedCount != 1 {
		t.Fatalf("pass:false verify step should count as located failure: %#v", report)
	}
	if report.StrategySummary.ByTaskType["unspecified"].FailureTypes["generated_freshness"] != 1 {
		t.Fatalf("expected generated_freshness failure type: %#v", report.StrategySummary)
	}
}

func TestAIQualityReportSummarizesTaskTypeLabelsAndStrategies(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "http-success.json", `{
  "id": "http-success",
  "source": "scenario",
  "task_type": "http_endpoint",
  "labels": ["single_service", "http"],
  "plan_pass": true,
  "apply_pass": true,
  "verify_pass": true
}`)
	writeFile(t, dir, "repair-evidence.json", `{
  "id": "repair-evidence",
  "source": "real_evidence",
  "kind": "nucleus.evidence_replay",
  "task_type": "generated_freshness",
  "labels": ["single_service", "repairable"],
  "evidence": {
    "kind": "nucleus.repair_evidence",
    "status": "needs_manual_action",
    "verification_pass": false,
    "failure_type": "generated_freshness",
    "manual_action_reason": "contract drift requires review",
    "rounds": [
      {
        "id": "repair-1",
        "strategy": "regenerate_generated_freshness",
        "status": "needs_manual_action"
      }
    ]
  }
}`)

	report := mustAIQualityReport(t, dir)
	byTaskType := report.StrategySummary.ByTaskType
	if byTaskType["http_endpoint"].TaskCount != 1 || byTaskType["http_endpoint"].SuccessRate != 1.0 {
		t.Fatalf("unexpected http_endpoint strategy summary: %#v", report.StrategySummary)
	}
	if byTaskType["generated_freshness"].SuccessCount != 0 || byTaskType["generated_freshness"].FailureTypes["generated_freshness"] != 1 {
		t.Fatalf("expected generated freshness failure summary: %#v", report.StrategySummary)
	}
	byLabel := report.StrategySummary.ByLabel
	if byLabel["single_service"].TaskCount != 2 || byLabel["single_service"].SuccessCount != 1 {
		t.Fatalf("unexpected single_service label summary: %#v", report.StrategySummary)
	}
	if report.StrategySummary.RepairStrategyCounts["regenerate_generated_freshness"] != 1 {
		t.Fatalf("expected repair strategy count: %#v", report.StrategySummary)
	}
	if report.StrategySummary.ManualActionReasonCounts["contract drift requires review"] != 1 {
		t.Fatalf("expected manual intervention reason count: %#v", report.StrategySummary)
	}
}

func TestAIQualityReportSummarizesCapabilityEvents(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "capability-evidence.json", `{
  "id": "capability-evidence",
  "source": "real_evidence",
  "kind": "nucleus.evidence_replay",
  "labels": ["single_service"],
  "evidence": {
    "kind": "nucleus.verify_result",
    "status": "failed",
    "verification_pass": false,
    "capability_events": [
      {
        "capability": "redis",
        "provider": "memory",
        "operation": "get",
        "status": "ok",
        "duration_ms": 1.2
      },
      {
        "capability": "mq",
        "provider": "kafka",
        "operation": "consume",
        "status": "failed"
      }
    ]
  }
}`)

	report := mustAIQualityReport(t, dir)
	if report.CapabilityEventCount != 2 {
		t.Fatalf("expected two capability events: %#v", report)
	}
	if report.CapabilityErrorCount != 1 {
		t.Fatalf("expected one capability error: %#v", report)
	}
	if report.CapabilitySummary["redis"].EventCount != 1 || report.CapabilitySummary["redis"].Operations["get"] != 1 {
		t.Fatalf("unexpected redis capability summary: %#v", report.CapabilitySummary)
	}
	if report.CapabilitySummary["mq"].ErrorCount != 1 || report.CapabilitySummary["mq"].Providers["kafka"] != 1 {
		t.Fatalf("unexpected mq capability summary: %#v", report.CapabilitySummary)
	}
}

func TestPlatformReadinessReportSummarizesServiceMetadata(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - http
ai:
  generated:
    - contract/gen
`)
	writeFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /healthz:
    get:
      operationId: getHealthz
`)
	writeFile(t, dir, "api/errors.yaml", `errors:
  - code: 0
    message: ok
    http_status: 200
`)

	report := mustPlatformReadinessReport(t, dir)
	if report.Service != "demo" || report.Version != "0.1.0" {
		t.Fatalf("unexpected service metadata: %#v", report)
	}
	if report.EndpointCount != 1 || report.CapabilityCount != 1 {
		t.Fatalf("unexpected counts: %#v", report)
	}
	if report.GeneratedFresh != false {
		t.Fatalf("missing generated marker should make generated_fresh false: %#v", report)
	}
}

func TestPlatformReadinessReportIncludesLocalArtifactsAndProviderStrategy(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - log
ai:
  generated:
    - contract/gen
`)
	writeFile(t, dir, "api/openapi.yaml", "openapi: 3.0.3\npaths: {}\n")
	writeFile(t, dir, "api/errors.yaml", "errors: []\n")
	writeFile(t, dir, "cmd/demo/main.go", `package main

import (
	_ "github.com/nucleuskit/bridge/zap"
	_ "github.com/nucleuskit/cap/log"
)

func main() {}
`)

	report := mustPlatformReadinessReport(t, dir)
	if report.PlatformUploadPayload.NetworkRequired || report.PlatformUploadPayload.Artifact != "local:artifacts/nucleus/platform-upload-payload.json" {
		t.Fatalf("platform payload should be local-only artifact metadata: %#v", report.PlatformUploadPayload)
	}
	if report.ReleaseDryRun.Artifact != "local:artifacts/nucleus/release-dry-run.json" {
		t.Fatalf("unexpected release dry-run artifact: %#v", report.ReleaseDryRun)
	}
	if len(report.ReadinessGates) == 0 {
		t.Fatalf("expected readiness gates: %#v", report)
	}
	if len(report.RiskGates) == 0 {
		t.Fatalf("expected risk gates: %#v", report)
	}
	if len(report.ProviderStrategy) != 1 {
		t.Fatalf("expected one provider strategy: %#v", report)
	}
	if report.ProviderStrategy[0].Capability != "log" || report.ProviderStrategy[0].Provider != "zap" {
		t.Fatalf("unexpected provider strategy identity: %#v", report.ProviderStrategy[0])
	}
	if report.ProviderStrategy[0].SDKStatus != "optional_external_provider_detected" {
		t.Fatalf("provider SDK should be recommendation state, not required default: %#v", report.ProviderStrategy[0])
	}
	if len(report.ProviderStrategy[0].Gaps) != 3 {
		t.Fatalf("expected health, fallback, and observability gaps: %#v", report.ProviderStrategy[0])
	}
}

func mustAIQualityReport(t *testing.T, dir string) aiQualityReport {
	t.Helper()
	result := buildAIQualityResult(dir, true)
	if !result.OK {
		t.Fatalf("buildAIQualityResult returned diagnostics: %#v", result.Diagnostics)
	}
	if result.AIQuality == nil {
		t.Fatal("AIQuality is nil")
	}
	return *result.AIQuality
}

func mustPlatformReadinessReport(t *testing.T, dir string) platformReadinessReport {
	t.Helper()
	result := buildPlatformReadinessResult(dir)
	if !result.OK {
		t.Fatalf("buildPlatformReadinessResult returned diagnostics: %#v", result.Diagnostics)
	}
	if result.PlatformReadiness == nil {
		t.Fatal("PlatformReadiness is nil")
	}
	return *result.PlatformReadiness
}

func writeFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}
