package report

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	decisioncmd "github.com/nucleuskit/nucleus/cmd/nucleus/internal/decision"
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
    "result_kind": "nucleus.repair_evidence",
    "ok": true,
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
    "result_kind": "nucleus.repair_evidence",
    "ok": false,
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
    "result_kind": "nucleus.verify_result",
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
        "kind": "generated_freshness",
        "status": "failed",
        "ok": false
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
    "result_kind": "nucleus.repair_evidence",
    "ok": false,
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
    "result_kind": "nucleus.verify_result",
    "ok": false,
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

func TestAIQualityReportSummarizesDecisionQualityAndRecipeCandidates(t *testing.T) {
	serviceDir := t.TempDir()
	tasksDir := filepath.Join(serviceDir, "artifacts", "nucleus", "ai-tasks")
	writeReportServiceScaffold(t, serviceDir)
	writeFile(t, serviceDir, ".nucleus/decisions/order-store.yaml", `schema_version: "decision.v1"
id: order-store-provider
capability: order_store
decision:
  provider: gorm
  library: gorm.io/gorm
  status: proposed
  locked: false
reason:
  - project needs an explicit provider decision
impact:
  files:
    - internal/order/store.go
verification:
  commands:
    - go test ./internal/order
`)
	acceptReportDecision(t, serviceDir, ".nucleus/decisions/order-store.yaml")
	writeFile(t, tasksDir, "plan-evidence.json", `{
  "id": "plan-evidence",
  "source": "scenario",
  "plan_pass": true,
  "apply_pass": true,
  "verify_pass": true,
  "evidence": {
    "result_kind": "nucleus.plan_result",
    "context": {
      "recipe_candidates": [
        {
          "id": "mysql-gorm",
          "decision_required": true
        },
        {
          "id": "mysql-sqlx",
          "decision_required": true
        }
      ]
    }
  }
}`)

	result := buildAIQualityResult(serviceDir, tasksDir, true)
	if !result.OK {
		t.Fatalf("buildAIQualityResult returned diagnostics: %#v", result.Diagnostics)
	}
	report := result.AIQuality
	if report == nil {
		t.Fatal("AIQuality is nil")
	}
	if report.DecisionQuality.Files != 1 || report.DecisionQuality.Valid != 1 || report.DecisionQuality.AcceptedLocked != 1 {
		t.Fatalf("decision_quality = %#v, want one accepted locked decision", report.DecisionQuality)
	}
	if report.RecipeCandidateUsage.CandidateCount != 2 || report.RecipeCandidateUsage.DecisionRequiredCount != 2 {
		t.Fatalf("recipe_candidate_usage = %#v, want two decision-required candidates", report.RecipeCandidateUsage)
	}
	if got := strings.Join(report.RecipeCandidateUsage.UniqueCandidateIDs, ","); got != "mysql-gorm,mysql-sqlx" {
		t.Fatalf("unique_candidate_ids = %q", got)
	}
	if result.Summary.DecisionFileCount != 1 || result.Summary.LockedDecisionCount != 1 || result.Summary.RecipeCandidateCount != 2 {
		t.Fatalf("summary = %#v, want decision and recipe candidate counts", result.Summary)
	}
}

func mustAIQualityReport(t *testing.T, dir string) aiQualityReport {
	t.Helper()
	result := buildAIQualityResult(dir, dir, true)
	if !result.OK {
		t.Fatalf("buildAIQualityResult returned diagnostics: %#v", result.Diagnostics)
	}
	if result.AIQuality == nil {
		t.Fatal("AIQuality is nil")
	}
	return *result.AIQuality
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

func writeReportServiceScaffold(t *testing.T, dir string) {
	t.Helper()
	writeFile(t, dir, "go.mod", "module example.com/demo\n\ngo 1.26.3\n")
	writeFile(t, dir, "demo.go", "package demo\n")
	writeFile(t, dir, "internal/order/store.go", "package order\n\ntype Store interface{}\n")
	writeFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - id: order_store
    kind: sql
ai:
  intent: test
  allowed_changes:
    - internal/**
    - .nucleus/**
`)
}

func acceptReportDecision(t *testing.T, dir string, path string) {
	t.Helper()
	cmd := decisioncmd.NewCommand(decisioncmd.Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"accept", path, "--by", "human", "--accepted-at", "2026-07-03T00:00:00Z", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("accept decision: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
}
