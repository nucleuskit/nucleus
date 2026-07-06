package verify

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nucleuskit/contract/inspect"
)

func TestCommandJSONSuccess(t *testing.T) {
	dir := t.TempDir()
	writeVerifyModule(t, dir)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute verify: %v\nstderr=%s\nstdout=%s", err, stderr.String(), stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var output struct {
		ResultKind    string          `json:"result_kind"`
		SchemaVersion string          `json:"schema_version"`
		SchemaRef     string          `json:"schema_ref"`
		OK            bool            `json:"ok"`
		Summary       verifySummary   `json:"summary"`
		Steps         []verifyStep    `json:"steps"`
		Diagnostics   json.RawMessage `json:"diagnostics"`
		Findings      json.RawMessage `json:"findings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if output.ResultKind != resultKindVerify {
		t.Fatalf("result_kind = %q, want %q", output.ResultKind, resultKindVerify)
	}
	if output.SchemaVersion != schemaVersion {
		t.Fatalf("schema_version = %q, want %q", output.SchemaVersion, schemaVersion)
	}
	if output.SchemaRef != schemaRef {
		t.Fatalf("schema_ref = %q, want %q", output.SchemaRef, schemaRef)
	}
	if !output.OK {
		t.Fatal("ok = false, want true")
	}
	if output.Summary.Failed != 0 {
		t.Fatalf("summary.failed = %d, want 0", output.Summary.Failed)
	}
	assertSuccessfulStepOrder(t, output.Steps, []string{"validate", "lint", "decision", "generated_freshness"})
	if output.Summary.Steps != 4 || output.Summary.Passed != 4 {
		t.Fatalf("summary = %#v, want 4 steps passed", output.Summary)
	}
}

func TestCommandJSONRunsManifestVerifyCommands(t *testing.T) {
	dir := t.TempDir()
	writeVerifyModuleWithCommands(t, dir, []string{"go test ./..."})

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute verify: %v\nstdout=%s", err, stdout.String())
	}

	var output struct {
		OK      bool          `json:"ok"`
		Summary verifySummary `json:"summary"`
		Steps   []verifyStep  `json:"steps"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if !output.OK {
		t.Fatal("ok = false, want true")
	}
	step, ok := findStep(output.Steps, "verify_command")
	if !ok {
		t.Fatalf("steps = %#v, want manifest verify command step", output.Steps)
	}
	if step.ID != "verify_command_1" || step.Command != "go test ./..." || !step.OK {
		t.Fatalf("verify command step = %#v", step)
	}
	if output.Summary.Steps != 5 || output.Summary.Passed != 5 {
		t.Fatalf("summary = %#v, want protocol steps plus one declared verify command", output.Summary)
	}
}

func TestCommandJSONValidationFailure(t *testing.T) {
	dir := t.TempDir()
	writeVerifyFile(t, dir, "go.mod", "module example.com/demo\n\ngo 1.26.3\n")
	writeVerifyFile(t, dir, "demo.go", "package demo\n")
	writeVerifyFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  version: "0.1.0"
ai:
  intent: test
capabilities: []
`)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	if !errors.Is(err, ErrVerifyFailed) {
		t.Fatalf("execute verify error = %v, want ErrVerifyFailed", err)
	}

	var output struct {
		OK          bool `json:"ok"`
		Diagnostics []struct {
			Code string `json:"code"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if output.OK {
		t.Fatal("ok = true, want false")
	}
	found := false
	for _, item := range output.Diagnostics {
		if item.Code == "manifest.service_name_required" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("diagnostics = %#v, want manifest.service_name_required", output.Diagnostics)
	}
}

func TestCommandJSONGeneratedFreshnessFailure(t *testing.T) {
	dir := t.TempDir()
	writeVerifyGeneratedModule(t, dir)
	writeVerifyFile(t, dir, "contract/gen/client.go", "package gen\n")

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	if !errors.Is(err, ErrVerifyFailed) {
		t.Fatalf("execute verify error = %v, want ErrVerifyFailed", err)
	}

	var output struct {
		OK       bool          `json:"ok"`
		Summary  verifySummary `json:"summary"`
		Steps    []verifyStep  `json:"steps"`
		Findings []struct {
			Rule string `json:"rule"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if output.OK {
		t.Fatal("ok = true, want false")
	}
	step, ok := findStep(output.Steps, "generated_freshness")
	if !ok {
		t.Fatalf("steps = %#v, want generated_freshness phase", output.Steps)
	}
	if step.OK {
		t.Fatalf("generated_freshness step OK = true, want false")
	}
	if step.Status != statusFailed {
		t.Fatalf("generated_freshness status = %q, want %q", step.Status, statusFailed)
	}
	if step.Output == "" {
		t.Fatalf("generated_freshness output is empty, want failure evidence")
	}
	if len(step.GeneratedFreshness) != 1 {
		t.Fatalf("generated_freshness = %#v, want one target", step.GeneratedFreshness)
	}
	if item := step.GeneratedFreshness[0]; item.Target != "contract/gen" || item.Fresh {
		t.Fatalf("generated_freshness item = %#v, want stale contract/gen", item)
	}
	if _, ok := findStep(output.Steps, "verify_command"); ok {
		t.Fatalf("steps = %#v, verify commands should not run after generated freshness failure", output.Steps)
	}
	foundL010 := false
	for _, finding := range output.Findings {
		if finding.Rule == "L010" {
			foundL010 = true
			break
		}
	}
	if !foundL010 {
		t.Fatalf("findings = %#v, want L010 freshness finding", output.Findings)
	}
	if output.Summary.Failed == 0 {
		t.Fatalf("summary.failed = 0, want failed steps")
	}
}

func TestCommandJSONDecisionFailure(t *testing.T) {
	dir := t.TempDir()
	writeVerifyModule(t, dir)
	writeVerifyFile(t, dir, ".nucleus/decisions/bad.yaml", `schema_version: "decision.v1"
id: bad
capability: missing
decision:
  provider: gorm
  status: proposed
  locked: false
reason:
  - missing capability declaration
verification:
  commands:
    - go test ./...
`)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--json"})

	err := cmd.Execute()
	if !errors.Is(err, ErrVerifyFailed) {
		t.Fatalf("execute verify error = %v, want ErrVerifyFailed", err)
	}

	var output struct {
		OK          bool         `json:"ok"`
		Steps       []verifyStep `json:"steps"`
		Diagnostics []struct {
			Code string `json:"code"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if output.OK {
		t.Fatal("ok = true, want false")
	}
	step, ok := findStep(output.Steps, "decision")
	if !ok {
		t.Fatalf("steps = %#v, want decision phase", output.Steps)
	}
	if step.OK || step.DecisionQuality == nil || step.DecisionQuality.Errors == 0 {
		t.Fatalf("decision step = %#v, want failed decision quality", step)
	}
	if _, ok := findStep(output.Steps, "generated_freshness"); ok {
		t.Fatalf("steps = %#v, generated_freshness should not run after decision failure", output.Steps)
	}
	found := false
	for _, item := range output.Diagnostics {
		if item.Code == "decision.capability_missing" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("diagnostics = %#v, want decision.capability_missing", output.Diagnostics)
	}
}

func TestCommandJSONGeneratedFreshnessSuccess(t *testing.T) {
	dir := t.TempDir()
	writeVerifyGeneratedModule(t, dir)
	writeVerifyFile(t, dir, "contract/gen/client.go", "package gen\n")
	sourceHash, err := inspect.ContractSourceHash(dir)
	if err != nil {
		t.Fatalf("ContractSourceHash(): %v", err)
	}
	writeVerifyFile(t, dir, "contract/gen/.nucleus-source.sha256", sourceHash+"\n")

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute verify: %v\nstdout=%s", err, stdout.String())
	}

	var output struct {
		OK    bool         `json:"ok"`
		Steps []verifyStep `json:"steps"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if !output.OK {
		t.Fatal("ok = false, want true")
	}
	step, ok := findStep(output.Steps, "generated_freshness")
	if !ok {
		t.Fatalf("steps = %#v, want generated_freshness phase", output.Steps)
	}
	if len(step.GeneratedFreshness) != 1 {
		t.Fatalf("generated_freshness = %#v, want one target", step.GeneratedFreshness)
	}
	if item := step.GeneratedFreshness[0]; item.Target != "contract/gen" || !item.Fresh || item.SourceHash != sourceHash {
		t.Fatalf("generated_freshness item = %#v, want fresh contract/gen", item)
	}
}

func TestCommandJSONDoesNotRunImplicitTidy(t *testing.T) {
	dir := t.TempDir()
	originalGoMod := `module example.com/demo

go 1.26.3

require example.com/unused v0.0.0

replace example.com/unused => ./unused
`
	writeVerifyFile(t, dir, "go.mod", originalGoMod)
	writeVerifyFile(t, dir, "demo.go", "package demo\n")
	writeVerifyFile(t, dir, "unused/go.mod", "module example.com/unused\n\ngo 1.26.3\n")
	writeVerifyFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
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
	cmd.SetArgs([]string{"--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute verify: %v\nstdout=%s", err, stdout.String())
	}

	var output struct {
		OK    bool         `json:"ok"`
		Steps []verifyStep `json:"steps"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if !output.OK {
		t.Fatal("ok = false, want true")
	}
	for _, forbidden := range []string{"tidy", "import", "build", "test"} {
		if _, ok := findStep(output.Steps, forbidden); ok {
			t.Fatalf("steps = %#v, verify should not run implicit %s step", output.Steps, forbidden)
		}
	}
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod after verify: %v", err)
	}
	if string(data) != originalGoMod {
		t.Fatalf("go.mod changed after verify:\n%s", data)
	}
}

func TestSanitizeCommandOutputRedactsSecretsPathsAndLongOutput(t *testing.T) {
	dir := t.TempDir()
	writeVerifyFile(t, dir, "go.mod", "module private.example.com/demo\n\ngo 1.26.3\n")
	raw := dir + "/demo.go: token=abc123 password: hunter2 Authorization: Bearer abc.def private.example.com/demo/internal\n" +
		strings.Repeat("x", maxCommandOutputRunes+16)

	output := sanitizeCommandOutput(raw, dir)
	for _, forbidden := range []string{dir, "abc123", "hunter2", "abc.def", "private.example.com/demo"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("sanitizeCommandOutput() leaked %q in %q", forbidden, output)
		}
	}
	for _, want := range []string{"token=[REDACTED]", "password: [REDACTED]", "Authorization: Bearer [REDACTED]", "<module>/internal", "[output truncated]"} {
		if !strings.Contains(output, want) {
			t.Fatalf("sanitizeCommandOutput() = %q, want %q", output, want)
		}
	}
}

func writeVerifyModule(t *testing.T, dir string) {
	t.Helper()
	writeVerifyFile(t, dir, "go.mod", "module example.com/demo\n\ngo 1.26.3\n")
	writeVerifyFile(t, dir, "demo.go", "package demo\n")
	writeVerifyFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  name: demo
  version: "0.1.0"
ai:
  intent: test
capabilities: []
`)
}

func writeVerifyModuleWithCommands(t *testing.T, dir string, commands []string) {
	t.Helper()
	writeVerifyFile(t, dir, "go.mod", "module example.com/demo\n\ngo 1.26.3\n")
	writeVerifyFile(t, dir, "demo.go", "package demo\n")
	var verifyBlock strings.Builder
	if len(commands) > 0 {
		verifyBlock.WriteString("verify:\n  commands:\n")
		for _, command := range commands {
			data, _ := json.Marshal(command)
			verifyBlock.WriteString("    - ")
			verifyBlock.Write(data)
			verifyBlock.WriteString("\n")
		}
	}
	writeVerifyFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  name: demo
  version: "0.1.0"
ai:
  intent: test
capabilities: []
`+verifyBlock.String())
}

func writeVerifyGeneratedModule(t *testing.T, dir string) {
	t.Helper()
	writeVerifyFile(t, dir, "go.mod", "module example.com/demo\n\ngo 1.26.3\n")
	writeVerifyFile(t, dir, "demo.go", "package demo\n")
	writeVerifyFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
info:
  title: demo
  version: 0.1.0
paths:
  /healthz:
    get:
      operationId: getHealthz
      responses:
        "204":
          description: ok
`)
	writeVerifyFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  name: demo
  version: "0.1.0"
ai:
  intent: test
  generated:
    - contract/gen
capabilities: []
`)
}

func writeVerifyFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func findStep(steps []verifyStep, phase string) (verifyStep, bool) {
	for _, step := range steps {
		if step.Phase == phase {
			return step, true
		}
	}
	return verifyStep{}, false
}

func assertSuccessfulStepOrder(t *testing.T, steps []verifyStep, want []string) {
	t.Helper()
	if len(steps) != len(want) {
		t.Fatalf("steps length = %d, want %d: %#v", len(steps), len(want), steps)
	}
	for index, phase := range want {
		step := steps[index]
		if step.Phase != phase {
			t.Fatalf("steps[%d].phase = %q, want %q", index, step.Phase, phase)
		}
		if step.ID != phase {
			t.Fatalf("steps[%d].id = %q, want %q", index, step.ID, phase)
		}
		if step.Kind != phase {
			t.Fatalf("steps[%d].kind = %q, want %q", index, step.Kind, phase)
		}
		if step.Sequence != index+1 {
			t.Fatalf("steps[%d].sequence = %d, want %d", index, step.Sequence, index+1)
		}
		if step.WorkingDir != "." {
			t.Fatalf("steps[%d].working_dir = %q, want .", index, step.WorkingDir)
		}
		if step.SchemaRef != schemaRef {
			t.Fatalf("steps[%d].schema_ref = %q, want %q", index, step.SchemaRef, schemaRef)
		}
		if step.Produces != resultKindVerify {
			t.Fatalf("steps[%d].produces = %q, want %q", index, step.Produces, resultKindVerify)
		}
		if step.Status != statusPassed || !step.OK || step.ExitCode != 0 {
			t.Fatalf("steps[%d] = %#v, want passed step", index, step)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
