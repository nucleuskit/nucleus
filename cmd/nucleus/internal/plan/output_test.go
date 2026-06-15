package plan

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildOutputAddsPlanMetadata(t *testing.T) {
	dir := newPlanFixture(t, []string{"api/**", "internal/domain/**", "internal/adapter/http/**"})

	output, err := BuildOutput(OutputOptions{
		Dir:  dir,
		Task: "新增 HTTP 接口",
	})
	if err != nil {
		t.Fatalf("BuildOutput() error = %v", err)
	}
	if output["result_kind"] != resultKindPlan {
		t.Fatalf("result_kind = %v, want %s", output["result_kind"], resultKindPlan)
	}
	if output["ok"] != true {
		t.Fatalf("ok = %v, want true", output["ok"])
	}
	summary, ok := output["summary"].(planSummary)
	if !ok {
		t.Fatalf("summary has type %T, want planSummary", output["summary"])
	}
	if summary.TaskType != taskTypeHTTPEndpoint {
		t.Fatalf("summary.task_type = %q, want %q", summary.TaskType, taskTypeHTTPEndpoint)
	}
	if !summary.ContractFirst {
		t.Fatal("summary.contract_first = false, want true")
	}
	if !containsString(anyStringSlice(output["commands"]), commandValidate) {
		t.Fatalf("commands = %#v, want validate command", output["commands"])
	}
}

func TestBuildOutputExecutableMarksBlockedEditsRequired(t *testing.T) {
	dir := newPlanFixture(t, []string{"docs/**"})

	output, err := BuildOutput(OutputOptions{
		Dir:        dir,
		Task:       "新增 HTTP 接口",
		Executable: true,
	})
	if err != nil {
		t.Fatalf("BuildOutput() error = %v", err)
	}
	if output["result_kind"] != resultKindExecutablePlan {
		t.Fatalf("result_kind = %v, want %s", output["result_kind"], resultKindExecutablePlan)
	}
	if output["ok"] != false {
		t.Fatalf("ok = %v, want false", output["ok"])
	}
	blocked, ok := output["blocked_edits"].([]map[string]any)
	if !ok {
		t.Fatalf("blocked_edits has type %T, want []map[string]any", output["blocked_edits"])
	}
	if len(blocked) == 0 {
		t.Fatal("blocked_edits = empty, want blocked edit")
	}
	if blocked[0]["required"] != true {
		t.Fatalf("blocked_edits[0].required = %v, want true", blocked[0]["required"])
	}
}

func TestCommandJSONBlockedReturnsSentinel(t *testing.T) {
	dir := newPlanFixture(t, []string{"docs/**"})
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--task", "新增 HTTP 接口", "--json"})

	err := cmd.Execute()
	if !errors.Is(err, ErrPlanBlocked) {
		t.Fatalf("execute plan error = %v, want ErrPlanBlocked", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty for JSON output", stderr.String())
	}

	var output struct {
		ResultKind   string      `json:"result_kind"`
		OK           bool        `json:"ok"`
		Summary      planSummary `json:"summary"`
		BlockedEdits []string    `json:"blocked_edits"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if output.ResultKind != resultKindPlan {
		t.Fatalf("result_kind = %q, want %q", output.ResultKind, resultKindPlan)
	}
	if output.OK {
		t.Fatal("ok = true, want false")
	}
	if output.Summary.BlockedEdits == 0 {
		t.Fatalf("summary.blocked_edits = 0, want blocked count; output=%s", stdout.String())
	}
	if len(output.BlockedEdits) == 0 {
		t.Fatalf("blocked_edits = empty, want blocked paths; output=%s", stdout.String())
	}
}

func TestCommandHumanSuccessOutput(t *testing.T) {
	dir := newPlanFixture(t, []string{"api/**", "internal/domain/**", "internal/adapter/http/**"})
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--task", "新增 HTTP 接口"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute plan: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	for _, want := range []string{
		"OK",
		"planned: http_endpoint",
		"contract first: true",
		"edits:",
		"commands:",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestCommandPrettyJSONOutput(t *testing.T) {
	dir := newPlanFixture(t, []string{"api/**", "internal/domain/**", "internal/adapter/http/**"})
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--task", "新增 HTTP 接口", "--json", "--pretty"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute plan: %v", err)
	}
	if !strings.Contains(stdout.String(), "\n  \"result_kind\"") {
		t.Fatalf("stdout = %q, want indented JSON", stdout.String())
	}
}

func newPlanFixture(t *testing.T, allowedChanges []string) string {
	t.Helper()
	dir := t.TempDir()
	manifest := `schema_version: "1.0"
service:
  name: fixture
  version: "0.1.0"
ai:
  intent: test
  allowed_changes:
`
	for _, path := range allowedChanges {
		manifest += "    - " + quoteYAMLString(path) + "\n"
	}
	manifest += `capabilities: []
nucleus: {}
`
	if err := os.WriteFile(filepath.Join(dir, "nucleus.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write nucleus.yaml: %v", err)
	}
	return dir
}

func quoteYAMLString(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
