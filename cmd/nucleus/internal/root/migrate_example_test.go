package root

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateCommandJSONRootWiring(t *testing.T) {
	dir := t.TempDir()
	writeRootMigrateFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
ai:
  intent: Exercise root migrate wiring.
  generated: []
`)
	writeRootMigrateFile(t, dir, "api/openapi.yaml", "openapi: 3.0.3\npaths: {}\n")
	writeRootMigrateFile(t, dir, "api/errors.yaml", "errors: []\n")

	cmd := New()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dir", dir, "migrate", "--from-version", "v0.1.0", "--to-version", "v0.2.0", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute migrate: %v\nstderr=%s", err, stderr.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode migrate output: %v\n%s", err, stdout.String())
	}
	assertString(t, output, "result_kind", "nucleus.migrate_result")
	assertString(t, output, "schema_ref", "contract/schema/migrate.schema.json")
	assertString(t, output, "mode", "plan")
	assertBool(t, output, "ok", true)
	summary := assertMap(t, output, "summary")
	assertString(t, summary, "service", "demo")
}

func writeRootMigrateFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}
