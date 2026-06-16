package apply

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandJSONDryRunSuccess(t *testing.T) {
	dir := t.TempDir()
	writeApplyService(t, dir, []string{"internal/domain/**"})
	planPath := filepath.Join(dir, "plan.json")
	writeApplyFile(t, dir, "plan.json", `{
  "edits": [
    {
      "path": "internal/domain/service.go",
      "content": "package domain\n"
    }
  ]
}`)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--json", "--plan", planPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute apply: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var output struct {
		Kind  string           `json:"kind"`
		Pass  bool             `json:"pass"`
		Mode  string           `json:"mode"`
		Steps []map[string]any `json:"steps"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if output.Kind != "nucleus.apply_evidence" || !output.Pass || output.Mode != "dry-run" {
		t.Fatalf("unexpected apply evidence: %#v", output)
	}
	if len(output.Steps) != 1 || output.Steps[0]["surface"] != "allowed" {
		t.Fatalf("steps = %#v, want one allowed edit check", output.Steps)
	}
}

func TestCommandHumanDryRunFailureReturnsSentinel(t *testing.T) {
	dir := t.TempDir()
	writeApplyService(t, dir, []string{"internal/domain/**"})
	planPath := filepath.Join(dir, "plan.json")
	writeApplyFile(t, dir, "plan.json", `{
  "edits": [
    {
      "path": "configs/prod.yaml",
      "content": "secret: value\n"
    }
  ]
}`)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--plan", planPath})

	err := cmd.Execute()
	if !errors.Is(err, ErrApplyFailed) {
		t.Fatalf("execute apply error = %v, want ErrApplyFailed", err)
	}
	if !strings.Contains(stderr.String(), "FAILED") {
		t.Fatalf("stderr = %q, want FAILED", stderr.String())
	}
	for _, want := range []string{"mode: dry-run", "steps: 1", "failed_steps: 1"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func writeApplyService(t *testing.T, dir string, allowed []string) {
	t.Helper()
	writeApplyFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities: []
ai:
  allowed_changes:
`+yamlApplyList(allowed)+`
`)
	writeApplyFile(t, dir, "api/openapi.yaml", "openapi: 3.0.3\npaths: {}\n")
	writeApplyFile(t, dir, "api/errors.yaml", "errors: []\n")
}

func yamlApplyList(values []string) string {
	var builder strings.Builder
	for _, value := range values {
		builder.WriteString("    - ")
		builder.WriteString(value)
		builder.WriteString("\n")
	}
	return builder.String()
}

func writeApplyFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}
