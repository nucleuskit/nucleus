package report

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReportCommandEmitsAIQualityJSON(t *testing.T) {
	serviceDir := t.TempDir()
	tasksDir := filepath.Join(serviceDir, "artifacts", "nucleus", "ai-tasks")
	writeReportFile(t, tasksDir, "scenario.json", `{
  "id": "scenario-success",
  "source": "scenario",
  "task_type": "http_endpoint",
  "labels": ["single_service", "http"],
  "plan_pass": true,
  "apply_pass": true,
  "verify_pass": true
}`)
	writeReportFile(t, tasksDir, "repair-evidence.json", `{
  "id": "missing-generated-real-evidence",
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

	dir := serviceDir
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--json", "--ai-tasks", tasksDir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute report: %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode report output: %v\n%s", err, stdout.String())
	}
	assertReportString(t, output, "result_kind", resultKindReport)
	assertReportString(t, output, "schema_version", schemaVersionReport)
	assertReportString(t, output, "schema_ref", schemaRefReport)
	assertReportString(t, output, "mode", reportModeAIQuality)
	assertReportBool(t, output, "ok", true)
	summary := assertReportMap(t, output, "summary")
	assertReportNumber(t, summary, "task_count", 2)
	assertReportNumber(t, summary, "scenario_task_count", 1)
	assertReportNumber(t, summary, "real_evidence_task_count", 1)
	assertReportNumber(t, summary, "repair_success_count", 1)
	aiQuality := assertReportMap(t, output, "ai_quality")
	assertReportString(t, aiQuality, "tasks_dir", "<external>/ai-tasks")
	if weekly, _ := aiQuality["weekly_report"].(string); !strings.Contains(weekly, "tasks=2") {
		t.Fatalf("weekly_report = %q, want tasks=2", weekly)
	}
	diagnostics := assertReportSlice(t, output, "diagnostics")
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want empty", diagnostics)
	}
}

func TestReportCommandRedactsAbsoluteAITaskPaths(t *testing.T) {
	serviceDir := t.TempDir()
	tasksDir := filepath.Join(serviceDir, "ai-tasks")
	writeReportFile(t, tasksDir, "broken.json", "{")

	dir := serviceDir
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--json", "--ai-tasks", tasksDir})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("execute report succeeded, want parse failure")
	}
	var output map[string]any
	if decodeErr := json.Unmarshal(stdout.Bytes(), &output); decodeErr != nil {
		t.Fatalf("decode report output: %v\n%s", decodeErr, stdout.String())
	}
	aiQuality := assertReportMap(t, output, "ai_quality")
	assertReportString(t, aiQuality, "tasks_dir", "<external>/ai-tasks")
	diagnostics := assertReportSlice(t, output, "diagnostics")
	for _, raw := range diagnostics {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		for _, key := range []string{"path", "message"} {
			text, _ := item[key].(string)
			if strings.Contains(text, serviceDir) {
				t.Fatalf("diagnostic %s leaked absolute path %q in %#v", key, serviceDir, item)
			}
		}
	}
}

func TestReportCommandDefaultHumanOutputDoesNotFailWhenDefaultTasksDirIsMissing(t *testing.T) {
	serviceDir := t.TempDir()
	dir := serviceDir
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute report: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "OK") || !strings.Contains(stdout.String(), "mode: ai_quality") {
		t.Fatalf("unexpected human stdout:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "tasks: 0") {
		t.Fatalf("expected zero-task summary in stdout:\n%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "report.ai_tasks_missing") {
		t.Fatalf("expected warning diagnostic in stderr, got:\n%s", stderr.String())
	}
}

func TestReportCommandExplicitMissingAITasksDirFailsWithJSONDiagnostics(t *testing.T) {
	serviceDir := t.TempDir()
	dir := serviceDir
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--json", "--ai-tasks", filepath.Join(serviceDir, "missing")})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("execute report succeeded, want failure")
	}
	if !errors.Is(err, ErrReportFailed) {
		t.Fatalf("error = %v, want ErrReportFailed", err)
	}
	var output map[string]any
	if decodeErr := json.Unmarshal(stdout.Bytes(), &output); decodeErr != nil {
		t.Fatalf("decode report output: %v\n%s", decodeErr, stdout.String())
	}
	assertReportBool(t, output, "ok", false)
	diagnostics := assertReportSlice(t, output, "diagnostics")
	assertReportContainsDiagnostic(t, diagnostics, "report.ai_tasks_read_failed")
}

func TestReportCommandRejectsRemovedPlatformFlag(t *testing.T) {
	serviceDir := t.TempDir()
	dir := serviceDir
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--platform", "--json"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("execute report succeeded, want unknown platform flag failure")
	}
	if !strings.Contains(err.Error(), "unknown flag: --platform") {
		t.Fatalf("error = %v, want unknown platform flag", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty output for rejected platform flag", stdout.String())
	}
}

func writeReportFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertReportMap(t *testing.T, value map[string]any, key string) map[string]any {
	t.Helper()
	item, ok := value[key].(map[string]any)
	if !ok {
		t.Fatalf("%s has type %T, want map[string]any", key, value[key])
	}
	return item
}

func assertReportSlice(t *testing.T, value map[string]any, key string) []any {
	t.Helper()
	item, ok := value[key].([]any)
	if !ok {
		t.Fatalf("%s has type %T, want []any", key, value[key])
	}
	return item
}

func assertReportString(t *testing.T, value map[string]any, key string, want string) {
	t.Helper()
	got, ok := value[key].(string)
	if !ok {
		t.Fatalf("%s has type %T, want string", key, value[key])
	}
	if got != want {
		t.Fatalf("%s = %q, want %q", key, got, want)
	}
}

func assertReportBool(t *testing.T, value map[string]any, key string, want bool) {
	t.Helper()
	got, ok := value[key].(bool)
	if !ok {
		t.Fatalf("%s has type %T, want bool", key, value[key])
	}
	if got != want {
		t.Fatalf("%s = %v, want %v", key, got, want)
	}
}

func assertReportNumber(t *testing.T, value map[string]any, key string, want float64) {
	t.Helper()
	got, ok := value[key].(float64)
	if !ok {
		t.Fatalf("%s has type %T, want number", key, value[key])
	}
	if got != want {
		t.Fatalf("%s = %v, want %v", key, got, want)
	}
}

func assertReportContainsDiagnostic(t *testing.T, diagnostics []any, code string) {
	t.Helper()
	for _, value := range diagnostics {
		item, ok := value.(map[string]any)
		if ok && item["code"] == code {
			return
		}
	}
	t.Fatalf("no diagnostic %q in %#v", code, diagnostics)
}
