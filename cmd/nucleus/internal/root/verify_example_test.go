package root

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyCommandJSONRootWiring(t *testing.T) {
	dir := t.TempDir()
	writeRootVerifyFile(t, dir, "go.mod", "module example.com/demo\n\ngo 1.26.3\n")
	writeRootVerifyFile(t, dir, "demo.go", "package demo\n")
	writeRootVerifyFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
ai:
  intent: test
capabilities: []
`)

	cmd := New()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dir", dir, "verify", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute verify: %v\nstderr=%s\nstdout=%s", err, stderr.String(), stdout.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode verify output: %v\n%s", err, stdout.String())
	}
	assertString(t, output, "result_kind", "nucleus.verify_result")
	assertBool(t, output, "ok", true)
}

func writeRootVerifyFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}
