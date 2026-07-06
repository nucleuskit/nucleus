package lint

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandJSONSuccess(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `schema_version: "2.0"
service:
  name: demo
  version: "0.1.0"
ai:
  intent: test
capabilities: []
`)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute lint: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var output struct {
		ResultKind    string      `json:"result_kind"`
		SchemaVersion string      `json:"schema_version"`
		SchemaRef     string      `json:"schema_ref"`
		OK            bool        `json:"ok"`
		Summary       lintSummary `json:"summary"`
		Diagnostics   []any       `json:"diagnostics"`
		Findings      []any       `json:"findings"`
		Strict        bool        `json:"strict"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if output.ResultKind != resultKindLint {
		t.Fatalf("result_kind = %q, want %q", output.ResultKind, resultKindLint)
	}
	if output.SchemaVersion != schemaVersionLint {
		t.Fatalf("schema_version = %q, want %q", output.SchemaVersion, schemaVersionLint)
	}
	if output.SchemaRef != schemaRefLint {
		t.Fatalf("schema_ref = %q, want %q", output.SchemaRef, schemaRefLint)
	}
	if !output.OK {
		t.Fatal("ok = false, want true")
	}
	if output.Diagnostics == nil {
		t.Fatal("diagnostics = nil, want empty array")
	}
	if output.Strict {
		t.Fatal("strict = true, want false")
	}
	if output.Summary.Findings != 0 {
		t.Fatalf("summary.findings = %d, want 0", output.Summary.Findings)
	}
	if output.Summary.Mode != "default" {
		t.Fatalf("summary.mode = %q, want default", output.Summary.Mode)
	}
	if !contains(output.Summary.Checked, "nucleus.yaml") {
		t.Fatalf("summary.checked = %#v, want nucleus.yaml", output.Summary.Checked)
	}
	if !contains(output.Summary.ActiveRules, "L006") {
		t.Fatalf("summary.active_rules = %#v, want L006", output.Summary.ActiveRules)
	}
	if output.Findings == nil {
		t.Fatal("findings = nil, want empty array")
	}
	if len(output.Findings) != 0 {
		t.Fatalf("findings = %#v, want empty", output.Findings)
	}
}

func TestCommandJSONFailureReturnsSentinel(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `schema_version: "2.0"
service:
  version: "0.1.0"
ai:
  intent: test
`)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	if !errors.Is(err, ErrLintFailed) {
		t.Fatalf("execute lint error = %v, want ErrLintFailed", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty for JSON output", stderr.String())
	}

	var output struct {
		OK       bool        `json:"ok"`
		Summary  lintSummary `json:"summary"`
		Findings []struct {
			Rule string `json:"rule"`
			Path string `json:"path"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if output.OK {
		t.Fatal("ok = true, want false")
	}
	if output.Summary.Findings == 0 {
		t.Fatalf("summary.findings = 0, want finding count; output=%s", stdout.String())
	}
	if len(output.Findings) == 0 {
		t.Fatalf("findings = empty, want L006 finding; output=%s", stdout.String())
	}
	if output.Findings[0].Rule != "L006" {
		t.Fatalf("first finding rule = %q, want L006", output.Findings[0].Rule)
	}
}

func TestCommandJSONStrictUsesStrictRules(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `schema_version: "2.0"
service:
  name: demo
  version: "0.1.0"
ai:
  intent: test
capabilities: []
dependencies:
  - name: upstream
    contract: deps/upstream.yaml
`)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--json", "--strict"})

	err := cmd.Execute()
	if !errors.Is(err, ErrLintFailed) {
		t.Fatalf("execute lint error = %v, want ErrLintFailed", err)
	}

	var output struct {
		Strict   bool `json:"strict"`
		Findings []struct {
			Rule string `json:"rule"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if !output.Strict {
		t.Fatal("strict = false, want true")
	}
	found := false
	for _, finding := range output.Findings {
		if finding.Rule == "L005" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("findings = %#v, want L005", output.Findings)
	}
}

func TestCommandHumanSuccessOutput(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `schema_version: "2.0"
service:
  name: demo
  version: "0.1.0"
ai:
  intent: test
capabilities: []
`)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute lint: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"OK",
		"linted: nucleus.yaml",
		"mode: default",
		"rules: L006, L008, L009, L011",
		"findings: 0",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("stdout = %q, want %q", output, want)
		}
	}
}

func TestCommandHumanFailureOutputMatchesValidateStyle(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `schema_version: "2.0"
service:
  version: "0.1.0"
ai:
  intent: test
`)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if !errors.Is(err, ErrLintFailed) {
		t.Fatalf("execute lint error = %v, want ErrLintFailed", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty on failure", stdout.String())
	}
	for _, want := range []string{"error nucleus.yaml L006:", "service.name is required"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr = %q, want %q", stderr.String(), want)
		}
	}
}

func TestCommandHumanStrictSuccessOutputIncludesStrictScope(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `schema_version: "2.0"
service:
  name: demo
  version: "0.1.0"
ai:
  intent: test
capabilities: []
`)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--strict"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute lint: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	for _, want := range []string{
		"mode: strict",
		"api/openapi.yaml",
		"rules: L006, L008, L009, L011, L001",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestCommandPrettyJSONOutput(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `schema_version: "2.0"
service:
  name: demo
  version: "0.1.0"
ai:
  intent: test
capabilities: []
`)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--json", "--pretty"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute lint: %v", err)
	}
	if !strings.Contains(stdout.String(), "\n  \"result_kind\"") {
		t.Fatalf("stdout = %q, want indented JSON", stdout.String())
	}
}

func writeManifest(t *testing.T, dir string, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "nucleus.yaml"), []byte(contents), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
