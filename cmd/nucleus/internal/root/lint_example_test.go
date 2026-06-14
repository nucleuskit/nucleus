package root

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLintCommandWithValidManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "nucleus.yaml"), []byte(`schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
ai:
  intent: test
capabilities: []
`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	cmd := New()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dir", dir, "lint", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute lint: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode lint output: %v\n%s", err, stdout.String())
	}
	assertString(t, output, "result_kind", "nucleus.lint_result")
	assertBool(t, output, "ok", true)
	summary := assertMap(t, output, "summary")
	assertNumber(t, summary, "findings", 0)
}
