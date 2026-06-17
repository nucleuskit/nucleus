package root

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReportCommandJSONRootWiring(t *testing.T) {
	dir := t.TempDir()
	writeRootReportFile(t, dir, "artifacts/nucleus/ai-tasks/scenario.json", `{
  "id": "scenario-success",
  "source": "scenario",
  "labels": ["single_service"],
  "plan_pass": true,
  "apply_pass": true,
  "verify_pass": true
}`)

	cmd := New()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dir", dir, "report", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute report: %v\nstderr=%s", err, stderr.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode report output: %v\n%s", err, stdout.String())
	}
	assertString(t, output, "result_kind", "nucleus.report_result")
	assertString(t, output, "schema_ref", "contract/schema/report.schema.json")
	assertString(t, output, "mode", "ai_quality")
	assertBool(t, output, "ok", true)
	summary := assertMap(t, output, "summary")
	assertNumber(t, summary, "task_count", 1)
}

func writeRootReportFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}
