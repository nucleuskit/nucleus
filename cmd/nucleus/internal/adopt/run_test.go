package adopt

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCreatesMinimalProtocolIndex(t *testing.T) {
	dir := t.TempDir()
	writeAdoptFile(t, dir, "go.mod", "module example.com/order-api\n\ngo 1.26.3\n")
	writeAdoptFile(t, dir, "service.go", `package order

type Store interface {
	Get(id string) error
}

func NewStore() Store {
	return nil
}
`)
	beforeGoMod := readAdoptFile(t, dir, "go.mod")

	output, err := run(Config{Dir: &dir}, &options{json: true})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !output.OK {
		t.Fatalf("output.OK = false: %#v", output.Diagnostics)
	}
	if output.ResultKind != resultKindAdopt || output.SchemaVersion != schemaVersionAdopt || output.SchemaRef != schemaRefAdopt {
		t.Fatalf("unexpected schema metadata: %#v", output)
	}
	if output.DetectedModule != "example.com/order-api" {
		t.Fatalf("detected module = %q", output.DetectedModule)
	}
	assertAdoptFileExists(t, dir, manifestFileName)
	assertAdoptFileExists(t, dir, decisionsKeepFile)
	assertAdoptFileExists(t, dir, nucleusReadmeFile)
	assertAdoptFileMissing(t, dir, "cmd")
	assertAdoptFileMissing(t, dir, "internal")
	assertAdoptFileMissing(t, dir, "deploy")
	assertAdoptFileMissing(t, dir, "go.sum")
	if got := readAdoptFile(t, dir, "go.mod"); got != beforeGoMod {
		t.Fatalf("go.mod was modified:\n%s", got)
	}
	manifest := readAdoptFile(t, dir, manifestFileName)
	for _, forbidden := range []string{"provider:", "driver:", "library:", "internal/", "go.mod", "go.sum"} {
		if strings.Contains(manifest, forbidden) {
			t.Fatalf("manifest leaked forbidden content %q:\n%s", forbidden, manifest)
		}
	}
	if !strings.Contains(manifest, `name: "order-api"`) {
		t.Fatalf("manifest did not infer service name:\n%s", manifest)
	}
	if output.Summary.CreatedFiles != 3 || output.Summary.Packages != 1 || output.Summary.Symbols == 0 {
		t.Fatalf("unexpected summary: %#v", output.Summary)
	}
}

func TestRunDetectsContractCandidatesWithoutRequiringContracts(t *testing.T) {
	dir := t.TempDir()
	writeAdoptFile(t, dir, "go.mod", "module example.com/library\n\ngo 1.26.3\n")
	writeAdoptFile(t, dir, "api/openapi.yaml", "openapi: 3.0.3\npaths: {}\n")
	writeAdoptFile(t, dir, "api/errors.yaml", "errors: []\n")

	output, err := run(Config{Dir: &dir}, &options{json: true})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if len(output.DetectedContracts) != 2 {
		t.Fatalf("detected contracts = %#v", output.DetectedContracts)
	}
	if len(output.DetectedTestCommands) == 0 {
		t.Fatalf("expected test command candidates: %#v", output)
	}
}

func TestRunAllowsProjectWithoutGoMod(t *testing.T) {
	dir := t.TempDir()

	output, err := run(Config{Dir: &dir}, &options{json: true})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !output.OK {
		t.Fatalf("output.OK = false: %#v", output.Diagnostics)
	}
	if len(output.Diagnostics) != 1 || output.Diagnostics[0].Code != "adopt.go_mod_missing" {
		t.Fatalf("diagnostics = %#v, want go_mod_missing warning", output.Diagnostics)
	}
	assertAdoptFileExists(t, dir, manifestFileName)
}

func TestRunCreatesMissingTargetDirectoryWithoutScanNoise(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing")

	output, err := run(Config{Dir: &dir}, &options{json: true})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !output.OK {
		t.Fatalf("output.OK = false: %#v", output.Diagnostics)
	}
	assertAdoptFileExists(t, dir, manifestFileName)
	for _, item := range output.Diagnostics {
		if item.Code == "adopt.scan_skipped" {
			t.Fatalf("unexpected scan diagnostic for missing target directory: %#v", output.Diagnostics)
		}
	}
}

func TestCommandRendersSchemaCompliantJSON(t *testing.T) {
	dir := t.TempDir()
	writeAdoptFile(t, dir, "go.mod", "module example.com/demo\n\ngo 1.26.3\n")
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--json", "--pretty", "--agent", "codex"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute adopt: %v", err)
	}
	var output result
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if output.ResultKind != resultKindAdopt || !output.OK {
		t.Fatalf("unexpected output: %#v", output)
	}
	assertAdoptFileExists(t, dir, codexInstructionFile)
}

func writeAdoptFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readAdoptFile(t *testing.T, dir string, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func assertAdoptFileExists(t *testing.T, dir string, name string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(name))); err != nil {
		t.Fatalf("%s should exist: %v", name, err)
	}
}

func assertAdoptFileMissing(t *testing.T, dir string, name string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(name))); err == nil {
		t.Fatalf("%s should not exist", name)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", name, err)
	}
}
