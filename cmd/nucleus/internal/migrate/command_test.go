package migrate

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateCommandEmitsJSONPlan(t *testing.T) {
	serviceDir := t.TempDir()
	writeMigrateService(t, serviceDir)

	dir := serviceDir
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--from-version", "v0.1.0", "--to-version", "v0.2.0", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute migrate: %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode migrate output: %v\n%s", err, stdout.String())
	}
	assertMigrateString(t, output, "result_kind", resultKindMigrate)
	assertMigrateString(t, output, "schema_version", schemaVersionMigrate)
	assertMigrateString(t, output, "schema_ref", schemaRefMigrate)
	assertMigrateBool(t, output, "ok", true)
	assertMigrateString(t, output, "mode", modePlan)
	summary := assertMigrateMap(t, output, "summary")
	assertMigrateString(t, summary, "service", "demo")
	assertMigrateString(t, summary, "compatibility", compatibilitySupported)
	assertMigrateNumber(t, summary, "steps", 6)
	assertMigrateNumber(t, summary, "contract_surfaces", 3)
	migration := assertMigrateMap(t, output, "migration")
	assertMigrateString(t, migration, "write_policy", writePolicyReport)
	commands := assertMigrateSlice(t, migration, "commands")
	if len(commands) < 3 {
		t.Fatalf("commands len = %d, want at least 3", len(commands))
	}
}

func TestMigrateCheckFailsOnStaleGeneratedTargets(t *testing.T) {
	serviceDir := t.TempDir()
	writeMigrateService(t, serviceDir)
	writeMigrateFile(t, serviceDir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
ai:
  intent: Exercise migrate checks.
  generated:
    - contract/gen
`)

	dir := serviceDir
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--from-version", "v0.1.0", "--to-version", "v0.2.0", "--check", "--json"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("execute migrate succeeded, want failure")
	}
	if !errors.Is(err, ErrMigrateFailed) {
		t.Fatalf("error = %v, want ErrMigrateFailed", err)
	}
	var output map[string]any
	if decodeErr := json.Unmarshal(stdout.Bytes(), &output); decodeErr != nil {
		t.Fatalf("decode migrate output: %v\n%s", decodeErr, stdout.String())
	}
	assertMigrateBool(t, output, "ok", false)
	diagnostics := assertMigrateSlice(t, output, "diagnostics")
	assertMigrateContainsDiagnostic(t, diagnostics, diagnosticGeneratedStale)
}

func TestMigrateCommandWritesReportForRelativePath(t *testing.T) {
	serviceDir := t.TempDir()
	writeMigrateService(t, serviceDir)

	dir := serviceDir
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--from-version", "v0.1.0", "--to-version", "v0.2.0", "--report", "artifacts/nucleus/migrate.json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute migrate: %v", err)
	}
	reportPath := filepath.Join(serviceDir, "artifacts", "nucleus", "migrate.json")
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var output map[string]any
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("decode report: %v\n%s", err, string(data))
	}
	assertMigrateString(t, output, "result_kind", resultKindMigrate)
	if !strings.Contains(stdout.String(), "OK") {
		t.Fatalf("human stdout = %q, want OK", stdout.String())
	}
}

func TestMigrateCommandRejectsMissingVersions(t *testing.T) {
	dir := t.TempDir()
	cmd := NewCommand(Config{Dir: &dir})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--to-version", "v0.2.0"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--from-version is required") {
		t.Fatalf("error = %v, want missing from-version", err)
	}
}

func TestMigrateCommandRejectsRelativeReportTraversal(t *testing.T) {
	serviceDir := t.TempDir()
	writeMigrateService(t, serviceDir)

	dir := serviceDir
	cmd := NewCommand(Config{Dir: &dir})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--from-version", "v0.1.0", "--to-version", "v0.2.0", "--report", "../migrate.json"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--report must resolve inside the service directory") {
		t.Fatalf("error = %v, want report traversal rejection", err)
	}
}

func TestMigrateCommandRejectsAbsoluteReportOutsideServiceDir(t *testing.T) {
	serviceDir := t.TempDir()
	writeMigrateService(t, serviceDir)
	outsidePath := filepath.Join(t.TempDir(), "migrate.json")

	dir := serviceDir
	cmd := NewCommand(Config{Dir: &dir})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--from-version", "v0.1.0", "--to-version", "v0.2.0", "--report", outsidePath})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--report must resolve inside the service directory") {
		t.Fatalf("error = %v, want absolute report rejection", err)
	}
}

func TestMigrateCommandRedactsInspectErrors(t *testing.T) {
	serviceDir := filepath.Join(t.TempDir(), "missing-service")
	dir := serviceDir
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--from-version", "v0.1.0", "--to-version", "v0.2.0", "--json"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("execute migrate succeeded, want inspect failure")
	}
	var output map[string]any
	if decodeErr := json.Unmarshal(stdout.Bytes(), &output); decodeErr != nil {
		t.Fatalf("decode migrate output: %v\n%s", decodeErr, stdout.String())
	}
	diagnostics := assertMigrateSlice(t, output, "diagnostics")
	assertMigrateContainsDiagnostic(t, diagnostics, diagnosticInspectFailed)
	for _, raw := range diagnostics {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		for _, key := range []string{"path", "message"} {
			text, _ := item[key].(string)
			if strings.Contains(text, serviceDir) {
				t.Fatalf("diagnostic %s leaked absolute path %q in %#v", key, serviceDir, item)
			}
		}
	}
}

func writeMigrateService(t *testing.T, dir string) {
	t.Helper()
	writeMigrateFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - log
ai:
  intent: Exercise migrate against a contract-first service.
  allowed_changes:
    - api/**
    - internal/**
    - contract/gen
  generated: []
nucleus:
  providers:
    log:
      provider: noop
`)
	writeMigrateFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /hello:
    get:
      operationId: sayHello
      responses:
        "200":
          description: ok
`)
	writeMigrateFile(t, dir, "api/errors.yaml", `errors:
  - code: 1001
    message: hello failed
    http_status: 500
`)
}

func writeMigrateFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertMigrateMap(t *testing.T, value map[string]any, key string) map[string]any {
	t.Helper()
	item, ok := value[key].(map[string]any)
	if !ok {
		t.Fatalf("%s has type %T, want map[string]any", key, value[key])
	}
	return item
}

func assertMigrateSlice(t *testing.T, value map[string]any, key string) []any {
	t.Helper()
	item, ok := value[key].([]any)
	if !ok {
		t.Fatalf("%s has type %T, want []any", key, value[key])
	}
	return item
}

func assertMigrateString(t *testing.T, value map[string]any, key string, want string) {
	t.Helper()
	got, ok := value[key].(string)
	if !ok {
		t.Fatalf("%s has type %T, want string", key, value[key])
	}
	if got != want {
		t.Fatalf("%s = %q, want %q", key, got, want)
	}
}

func assertMigrateBool(t *testing.T, value map[string]any, key string, want bool) {
	t.Helper()
	got, ok := value[key].(bool)
	if !ok {
		t.Fatalf("%s has type %T, want bool", key, value[key])
	}
	if got != want {
		t.Fatalf("%s = %v, want %v", key, got, want)
	}
}

func assertMigrateNumber(t *testing.T, value map[string]any, key string, want float64) {
	t.Helper()
	got, ok := value[key].(float64)
	if !ok {
		t.Fatalf("%s has type %T, want number", key, value[key])
	}
	if got != want {
		t.Fatalf("%s = %v, want %v", key, got, want)
	}
}

func assertMigrateContainsDiagnostic(t *testing.T, diagnostics []any, code string) {
	t.Helper()
	for _, value := range diagnostics {
		item, ok := value.(map[string]any)
		if ok && item["code"] == code {
			return
		}
	}
	t.Fatalf("no diagnostic %q in %#v", code, diagnostics)
}
