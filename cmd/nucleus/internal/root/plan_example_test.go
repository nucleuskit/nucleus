package root

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestPlanCommandWithHelloHTTPExample(t *testing.T) {
	exampleDir := writeRootExampleService(t)

	cmd := New()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dir", exampleDir, "plan", "--task", "新增 HTTP 接口", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute plan: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode plan output: %v\n%s", err, stdout.String())
	}
	assertString(t, output, "result_kind", "nucleus.plan_result")
	assertBool(t, output, "ok", true)
	assertString(t, output, "task_type", "http_endpoint")
	summary := assertMap(t, output, "summary")
	assertNumber(t, summary, "blocked_edits", 0)
}
