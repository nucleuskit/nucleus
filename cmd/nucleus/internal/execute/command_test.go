package execute

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestCommandJSONBlockedReturnsSentinel(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "plan.json", `{
  "commands": [
    {
      "id": "cmd-1",
      "command": "nucleus lint --dir .",
      "working_dir": ".",
      "timeout_seconds": 5
    }
  ]
}`)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--json", "--plan", dir + "/plan.json", "--allow-command", "go"})

	err := cmd.Execute()
	if !errors.Is(err, ErrExecuteFailed) {
		t.Fatalf("execute command error = %v, want ErrExecuteFailed", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty for JSON output", stderr.String())
	}

	var output struct {
		Kind   string           `json:"kind"`
		Pass   bool             `json:"pass"`
		Status string           `json:"status"`
		Steps  []map[string]any `json:"steps"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if output.Kind != "nucleus.executor_evidence" || output.Pass || output.Status != "failed" {
		t.Fatalf("unexpected executor evidence: %#v", output)
	}
	if len(output.Steps) != 1 || output.Steps[0]["status"] != "blocked" {
		t.Fatalf("steps = %#v, want blocked command", output.Steps)
	}
}

func TestCommandHumanSuccessOutput(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "plan.json", `{
  "commands": [
    {
      "id": "cmd-1",
      "command": "go version",
      "working_dir": ".",
      "timeout_seconds": 5
    }
  ]
}`)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--plan", dir + "/plan.json", "--allow-command", "go"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	for _, want := range []string{"OK", "status: passed", "steps: 1", "exit_codes: 1"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}
