package capability

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	contractlint "github.com/nucleuskit/contract/lint"
	"github.com/nucleuskit/contract/validation"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/initcmd"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/plan"
)

func TestAddRedisProducesVerifiableService(t *testing.T) {
	dir := initService(t)

	output := executeCapabilityCommand(t, dir, "add", "redis", "--json")
	if output.ResultKind != resultKindCapability || !output.OK {
		t.Fatalf("unexpected output: %#v", output)
	}
	if output.Capability != "redis" || output.Provider != "redis" {
		t.Fatalf("capability/provider = %s/%s, want redis/redis", output.Capability, output.Provider)
	}
	assertChange(t, output.Files, "nucleus.yaml", actionUpdated)
	assertChange(t, output.Files, "internal/component/redis/redis.go", actionCreated)
	assertFileContains(t, dir, "nucleus.yaml", "redis:")
	assertFileContains(t, dir, "nucleus.yaml", "provider: redis")
	assertFileContains(t, dir, "internal/component/redis/redis.go", "func NewFromEnv() (*RedisComponent, error)")
	assertFileContains(t, dir, "internal/app/capabilities_redis.go", "func NewRedisCapability() (*RedisCapability, error)")
	assertValidationClean(t, dir)
	assertStrictLintClean(t, dir)
	runGoTest(t, dir)
}

func TestAddSQLPostgresProducesBuildableService(t *testing.T) {
	dir := initService(t)

	output := executeCapabilityCommand(t, dir, "add", "sql", "--provider", "postgres", "--json")
	if output.Capability != "sql" || output.Provider != "postgres" || !output.OK {
		t.Fatalf("unexpected output: %#v", output)
	}
	assertChange(t, output.Files, "go.mod", actionUpdated)
	assertChange(t, output.Files, "internal/component/sql/postgres.go", actionCreated)
	assertFileContains(t, dir, "go.mod", "github.com/lib/pq v1.10.9")
	assertFileNotContains(t, dir, "go.mod", "anniext.cn")
	assertFileContains(t, dir, "internal/component/sql/postgres.go", `import (`)
	assertFileContains(t, dir, "internal/component/sql/postgres.go", `_ "github.com/lib/pq"`)
	assertFileNotContains(t, dir, "internal/component/sql/postgres.go", "anniext.cn")
	assertValidationClean(t, dir)
	assertStrictLintClean(t, dir)
	runGoTest(t, dir)
}

func TestDryRunDoesNotWriteFiles(t *testing.T) {
	dir := initService(t)
	before := readFile(t, dir, "nucleus.yaml")

	output := executeCapabilityCommand(t, dir, "add", "redis", "--dry-run", "--json")
	if !output.OK || !output.DryRun {
		t.Fatalf("unexpected dry-run output: %#v", output)
	}
	assertChange(t, output.Files, "nucleus.yaml", actionWouldUpdate)
	assertChange(t, output.Files, "internal/component/redis/redis.go", actionWouldCreate)
	assertFileNotExists(t, dir, "internal/component/redis/redis.go")
	if after := readFile(t, dir, "nucleus.yaml"); after != before {
		t.Fatalf("nucleus.yaml changed during dry-run\nbefore=%s\nafter=%s", before, after)
	}
}

func TestRejectsGeneratedFileConflictUnlessForced(t *testing.T) {
	dir := initService(t)
	writeFile(t, dir, "internal/component/redis/redis.go", "package redis\n\nconst custom = true\n")
	before := readFile(t, dir, "nucleus.yaml")

	output, err := executeCapabilityCommandError(t, dir, "add", "redis", "--json")
	if !errors.Is(err, ErrCapabilityFailed) {
		t.Fatalf("execute error = %v, want ErrCapabilityFailed", err)
	}
	if output.OK {
		t.Fatalf("ok = true, want false: %#v", output)
	}
	assertChange(t, output.Files, "internal/component/redis/redis.go", actionConflict)
	if after := readFile(t, dir, "nucleus.yaml"); after != before {
		t.Fatalf("nucleus.yaml changed despite conflict\nbefore=%s\nafter=%s", before, after)
	}

	forced := executeCapabilityCommand(t, dir, "add", "redis", "--force", "--json")
	if !forced.OK || !forced.Forced {
		t.Fatalf("unexpected forced output: %#v", forced)
	}
	assertFileContains(t, dir, "internal/component/redis/redis.go", "type RedisConfig struct")
	assertValidationClean(t, dir)
	assertStrictLintClean(t, dir)
	runGoTest(t, dir)
}

func TestRejectsUnsupportedProviderWithStructuredJSON(t *testing.T) {
	dir := initService(t)

	output, err := executeCapabilityCommandError(t, dir, "add", "redis", "--provider", "postgres", "--json")
	if !errors.Is(err, ErrCapabilityFailed) {
		t.Fatalf("execute error = %v, want ErrCapabilityFailed", err)
	}
	if output.OK {
		t.Fatalf("ok = true, want false: %#v", output)
	}
	if len(output.Diagnostics) != 1 || output.Diagnostics[0].Code != "capability.provider_unsupported" {
		t.Fatalf("diagnostics = %#v, want provider unsupported", output.Diagnostics)
	}
}

func TestRejectsRuntimeCapabilityScaffold(t *testing.T) {
	dir := initService(t)

	output, err := executeCapabilityCommandError(t, dir, "add", "http", "--json")
	if !errors.Is(err, ErrCapabilityFailed) {
		t.Fatalf("execute error = %v, want ErrCapabilityFailed", err)
	}
	if output.OK {
		t.Fatalf("ok = true, want false: %#v", output)
	}
	if len(output.Diagnostics) != 1 || output.Diagnostics[0].Code != "capability.runtime_unsupported" {
		t.Fatalf("diagnostics = %#v, want runtime unsupported", output.Diagnostics)
	}
}

func TestPrettyJSONOutput(t *testing.T) {
	dir := initService(t)
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"add", "redis", "--dry-run", "--json", "--pretty"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute capability: %v", err)
	}
	if !strings.Contains(stdout.String(), "\n  \"result_kind\"") {
		t.Fatalf("stdout = %q, want indented JSON", stdout.String())
	}
}

func TestPlanSuggestedCapabilityCommandsAreAcceptedByDryRun(t *testing.T) {
	dir := initService(t)
	for _, task := range []string{
		"接入 redis 能力",
		"接入 postgres 数据库能力",
		"接入 sentry 错误追踪能力",
	} {
		output, err := plan.BuildOutput(plan.OutputOptions{Dir: dir, Task: task})
		if err != nil {
			t.Fatalf("BuildOutput(%q): %v", task, err)
		}
		commands := anyStringSlice(output["commands"])
		found := false
		for _, command := range commands {
			if !strings.HasPrefix(command, "nucleus capability add ") {
				continue
			}
			found = true
			args := append(strings.Fields(strings.TrimPrefix(command, "nucleus capability ")), "--dry-run", "--json")
			result := executeCapabilityCommand(t, dir, args...)
			if !result.OK {
				t.Fatalf("dry-run for %q returned not ok: %#v", command, result)
			}
		}
		if !found {
			t.Fatalf("task %q produced no capability add command: %#v", task, commands)
		}
	}
}

type capabilityCommandOutput struct {
	ResultKind  string `json:"result_kind"`
	OK          bool   `json:"ok"`
	Capability  string `json:"capability"`
	Provider    string `json:"provider"`
	DryRun      bool   `json:"dry_run,omitempty"`
	Forced      bool   `json:"forced,omitempty"`
	Files       []fileChange
	Diagnostics []struct {
		Severity string `json:"severity"`
		Code     string `json:"code"`
		Path     string `json:"path,omitempty"`
		Message  string `json:"message"`
	} `json:"diagnostics,omitempty"`
}

func initService(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := initcmd.NewCommand(initcmd.Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--name", "demo", "--module", "example.com/demo"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute init: %v\nstderr=%s\nstdout=%s", err, stderr.String(), stdout.String())
	}
	return dir
}

func executeCapabilityCommand(t *testing.T, dir string, args ...string) capabilityCommandOutput {
	t.Helper()
	output, err := executeCapabilityCommandError(t, dir, args...)
	if err != nil {
		t.Fatalf("execute capability: %v\noutput=%#v", err, output)
	}
	return output
}

func executeCapabilityCommandError(t *testing.T, dir string, args ...string) (capabilityCommandOutput, error) {
	t.Helper()
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	var output capabilityCommandOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	return output, err
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

func assertChange(t *testing.T, files []fileChange, path string, action string) {
	t.Helper()
	for _, file := range files {
		if file.Path == path {
			if file.Action != action {
				t.Fatalf("file %s action = %s, want %s; files=%#v", path, file.Action, action, files)
			}
			return
		}
	}
	t.Fatalf("files = %#v, want %s", files, path)
}

func assertFileContains(t *testing.T, dir string, name string, want string) {
	t.Helper()
	data := readFile(t, dir, name)
	if !strings.Contains(data, want) {
		t.Fatalf("%s = %q, want %q", name, data, want)
	}
}

func assertFileNotContains(t *testing.T, dir string, name string, unwanted string) {
	t.Helper()
	data := readFile(t, dir, name)
	if strings.Contains(data, unwanted) {
		t.Fatalf("%s = %q, did not want %q", name, data, unwanted)
	}
}

func assertFileNotExists(t *testing.T, dir string, name string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, name)); err == nil || !os.IsNotExist(err) {
		t.Fatalf("%s exists or stat failed with %v, want not exist", name, err)
	}
}

func readFile(t *testing.T, dir string, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func writeFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func anyStringSlice(value any) []string {
	items, ok := value.([]string)
	if ok {
		return items
	}
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		text, ok := item.(string)
		if ok {
			values = append(values, text)
		}
	}
	return values
}
