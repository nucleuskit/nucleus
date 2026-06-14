package root

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateCommandWithHelloHTTPExample(t *testing.T) {
	repoRoot := repositoryRoot(t)
	exampleDir := filepath.Join(repoRoot, "example", "hello-http")

	cmd := New()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--dir", exampleDir, "validate", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute validate: %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode validate output: %v\n%s", err, stdout.String())
	}
	assertString(t, output, "result_kind", "nucleus.validate_result")
	assertBool(t, output, "ok", true)
	summary := assertMap(t, output, "summary")
	assertNumber(t, summary, "errors", 0)
	assertNumber(t, summary, "warnings", 0)
	assertContainsString(t, assertSlice(t, summary, "checked"), "nucleus.yaml")
}

func TestValidateCommandReportsDiagnostics(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "nucleus.yaml"), []byte(`schema_version: "1.0"
service:
  version: "0.1.0"
ai:
  intent: test
`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	cmd := New()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--dir", dir, "validate", "--json"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("execute validate succeeded, want failure")
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode validate output: %v\n%s", err, stdout.String())
	}
	assertBool(t, output, "ok", false)
	diagnostics := assertSlice(t, output, "diagnostics")
	assertContainsFact(t, diagnostics, "code", "manifest.service_name_required")
}

func assertBool(t *testing.T, value map[string]any, key string, want bool) {
	t.Helper()
	got, ok := value[key].(bool)
	if !ok {
		t.Fatalf("%s has type %T, want bool", key, value[key])
	}
	if got != want {
		t.Fatalf("%s = %v, want %v", key, got, want)
	}
}

func assertNumber(t *testing.T, value map[string]any, key string, want float64) {
	t.Helper()
	got, ok := value[key].(float64)
	if !ok {
		t.Fatalf("%s has type %T, want number", key, value[key])
	}
	if got != want {
		t.Fatalf("%s = %v, want %v", key, got, want)
	}
}

func assertContainsString(t *testing.T, values []any, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("no string %q in %#v", want, values)
}
