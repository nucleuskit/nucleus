package root

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestScenarioCommandWithHelloHTTPExample(t *testing.T) {
	repoRoot := repositoryRoot(t)
	exampleDir := filepath.Join(repoRoot, "example", "hello-http")

	cmd := New()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--dir", exampleDir, "scenario", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute scenario: %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode scenario output: %v\n%s", err, stdout.String())
	}
	assertString(t, output, "result_kind", "nucleus.scenario_plan_result")
	assertString(t, output, "kind", "nucleus.scenario_plan")
	scenarios := assertSlice(t, output, "scenarios")
	if len(scenarios) == 0 {
		t.Fatalf("expected at least one scenario: %#v", output)
	}
}
