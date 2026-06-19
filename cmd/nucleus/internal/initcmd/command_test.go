package initcmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	contractlint "github.com/nucleuskit/contract/lint"
	"github.com/nucleuskit/contract/validation"
)

func TestCommandServiceTemplateProducesVerifiableProject(t *testing.T) {
	dir := t.TempDir()

	output := executeInitCommand(t, dir, "--name", "demo", "--module", "example.com/demo", "--template", "service")
	if output.ResultKind != resultKindInit || !output.OK || output.Template != "service" {
		t.Fatalf("unexpected output: %#v", output)
	}
	assertContains(t, output.Files, "nucleus.yaml")
	assertContains(t, output.Files, "contract/gen/.nucleus-source.sha256")
	assertContains(t, output.Files, "internal/adapter/http/gen/.nucleus-source.sha256")
	assertFileContains(t, dir, "go.mod", "github.com/nucleuskit/http v0.1.0-alpha.1.0.20260615170339-225ca98f40d7")
	assertFileNotContains(t, dir, "go.mod", "replace ")
	assertFileNotContains(t, dir, "api/errors.yaml", "code: 0")
	assertFileContains(t, dir, "internal/adapter/http/gen/routes.gen.go", `runtimehttp "github.com/nucleuskit/http"`)
	assertValidationClean(t, dir)
	assertStrictLintClean(t, dir)
	runGoTest(t, dir)
}

func TestCommandWorkerTemplateProducesVerifiableProject(t *testing.T) {
	dir := t.TempDir()

	output := executeInitCommand(t, dir, "--name", "jobs", "--module", "example.com/jobs", "--template", "worker", "--json")
	if output.ResultKind != resultKindInit || !output.OK || output.Template != "worker" {
		t.Fatalf("unexpected output: %#v", output)
	}
	assertContains(t, output.Files, "cmd/jobs/main.go")
	assertContains(t, output.Files, "internal/worker/handler.go")
	assertFileNotExists(t, dir, "go.sum")
	assertFileNotExists(t, dir, "api/openapi.yaml")
	assertFileNotExists(t, dir, "api/errors.yaml")
	assertFileContains(t, dir, "nucleus.yaml", "provider: local")
	assertValidationClean(t, dir)
	assertStrictLintClean(t, dir)
	runGoTest(t, dir)
}

func TestCommandLibraryTemplateProducesVerifiableProject(t *testing.T) {
	dir := t.TempDir()

	output := executeInitCommand(t, dir, "--name", "util-lib", "--module", "example.com/util-lib", "--template", "library", "--json")
	if output.ResultKind != resultKindInit || !output.OK || output.Template != "library" {
		t.Fatalf("unexpected output: %#v", output)
	}
	assertContains(t, output.Files, "util_lib.go")
	assertFileNotExists(t, dir, "api/openapi.yaml")
	assertFileNotExists(t, dir, "api/errors.yaml")
	assertFileContains(t, dir, "nucleus.yaml", "capabilities: []")
	assertValidationClean(t, dir)
	assertStrictLintClean(t, dir)
	runGoTest(t, dir)
}

func TestCommandRejectsUnsafeInputs(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("occupied\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := NewCommand(Config{Dir: &dir})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--name", "demo", "--module", "example.com/demo"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "target directory is not empty") {
		t.Fatalf("execute error = %v, want non-empty directory failure", err)
	}

	empty := t.TempDir()
	cmd = NewCommand(Config{Dir: &empty})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--name", "../demo", "--module", "example.com/demo"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "service name") {
		t.Fatalf("execute error = %v, want service name validation failure", err)
	}

	for _, module := range []string{
		"example.com/demo@v1",
		"example.com/demo\"bad",
		"example.com//demo",
		"example.com/../demo",
		"example.com/demo!",
	} {
		empty := t.TempDir()
		cmd = NewCommand(Config{Dir: &empty})
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"--name", "demo", "--module", module})
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "module path") {
			t.Fatalf("execute with module %q error = %v, want module path validation failure", module, err)
		}
	}
}

func TestCommandDefaultFailureWritesStructuredJSON(t *testing.T) {
	dir := t.TempDir()
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--template", "service"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--name is required") {
		t.Fatalf("execute error = %v, want missing name", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty for structured output", stderr.String())
	}
	var output initCommandOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if !strings.Contains(stdout.String(), `"files":[]`) {
		t.Fatalf("stdout = %q, want empty files array", stdout.String())
	}
	if output.OK {
		t.Fatalf("ok = true, want false: %#v", output)
	}
	if output.ResultKind != resultKindInit {
		t.Fatalf("result_kind = %q, want %q", output.ResultKind, resultKindInit)
	}
	if len(output.Diagnostics) != 1 || output.Diagnostics[0].Code != "init.name_required" {
		t.Fatalf("diagnostics = %#v, want init.name_required", output.Diagnostics)
	}
}

func TestCommandHumanFailureWritesError(t *testing.T) {
	dir := t.TempDir()
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--template", "service", "--human"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--name is required") {
		t.Fatalf("execute error = %v, want missing name", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	for _, want := range []string{"FAILED", "--name is required"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr = %q, want %q", stderr.String(), want)
		}
	}
}

func TestCommandPrettyJSONOutput(t *testing.T) {
	dir := t.TempDir()

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--name", "demo", "--module", "example.com/demo", "--pretty"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute init: %v", err)
	}
	if !strings.Contains(stdout.String(), "\n  \"result_kind\"") {
		t.Fatalf("stdout = %q, want indented JSON", stdout.String())
	}
}

type initCommandOutput struct {
	ResultKind  string `json:"result_kind"`
	OK          bool   `json:"ok"`
	Template    string `json:"template"`
	ServiceName string `json:"service_name"`
	Module      string `json:"module"`
	Files       []string
	Generated   []string `json:"generated,omitempty"`
	Diagnostics []struct {
		Severity string `json:"severity"`
		Code     string `json:"code"`
		Message  string `json:"message"`
	} `json:"diagnostics,omitempty"`
}

func executeInitCommand(t *testing.T, dir string, args ...string) initCommandOutput {
	t.Helper()
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute init: %v\nstderr=%s\nstdout=%s", err, stderr.String(), stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	var output initCommandOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	return output
}

func assertValidationClean(t *testing.T, dir string) {
	t.Helper()
	diagnostics := validation.ValidateService(dir)
	if diagnostics.Failed() {
		t.Fatalf("validation diagnostics = %#v", diagnostics)
	}
}

func assertStrictLintClean(t *testing.T, dir string) {
	t.Helper()
	findings := contractlint.Run(dir, true)
	if len(findings) != 0 {
		t.Fatalf("strict lint findings = %#v", findings)
	}
}

func runGoTest(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go test ./... failed: %v\n%s", err, string(output))
	}
}

func assertContains(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("values = %#v, want %q", values, want)
}

func assertFileContains(t *testing.T, dir string, name string, want string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("%s = %q, want %q", name, string(data), want)
	}
}

func assertFileNotContains(t *testing.T, dir string, name string, unwanted string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), unwanted) {
		t.Fatalf("%s = %q, did not want %q", name, string(data), unwanted)
	}
}

func assertFileNotExists(t *testing.T, dir string, name string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, name)); err == nil || !os.IsNotExist(err) {
		t.Fatalf("%s exists or stat failed with %v, want not exist", name, err)
	}
}
