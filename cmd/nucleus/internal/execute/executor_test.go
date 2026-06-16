package execute

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExecutePlanCommandsAllowsCommandSuccess(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "plan.json", `{
  "kind": "nucleus.executable_plan",
  "commands": [
    {
      "id": "cmd-1",
      "command": "go version",
      "cwd": ".",
      "timeout_seconds": 5
    }
  ],
  "assertions": [
    {
      "id": "assert-cmd-1",
      "type": "command_exit",
      "target": "go version",
      "expected": "exit_code == 0"
    }
  ],
  "rollback": [
    {
      "id": "rollback-1",
      "target": "none",
      "strategy": "no writes performed",
      "required": false
    }
  ]
}`)

	evidence, err := ExecutePlanCommands(dir, filepath.Join(dir, "plan.json"), []string{"go"})
	if err != nil {
		t.Fatalf("ExecutePlanCommands returned error: %v", err)
	}
	if evidence["kind"] != "nucleus.executor_evidence" || evidence["pass"] != true || evidence["status"] != "passed" {
		t.Fatalf("unexpected passing evidence: %#v", evidence)
	}
	steps := evidence["steps"].([]map[string]any)
	if len(steps) != 1 || steps[0]["command"] != "go version" || steps[0]["exit_code"] != 0 {
		t.Fatalf("unexpected command step: %#v", steps)
	}
	exitCodes := evidence["exit_codes"].([]map[string]any)
	if len(exitCodes) != 1 || exitCodes[0]["exit_code"] != 0 {
		t.Fatalf("unexpected exit code summary: %#v", exitCodes)
	}
}

func TestExecutePlanCommandsRejectsCommandOutsideAllowlist(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "plan.json", `{
  "commands": [
    {
      "id": "cmd-1",
      "command": "nucleus lint --dir .",
      "working_dir": ".",
      "timeout_seconds": 5
    }
  ]
}`)

	evidence, err := ExecutePlanCommands(dir, filepath.Join(dir, "plan.json"), []string{"go"})
	if err != nil {
		t.Fatalf("allowlist rejection should be represented as evidence, got error: %v", err)
	}
	if evidence["pass"] != false || evidence["status"] != "failed" {
		t.Fatalf("unexpected rejection evidence: %#v", evidence)
	}
	steps := evidence["steps"].([]map[string]any)
	if steps[0]["status"] != "blocked" || steps[0]["exit_code"] != 126 {
		t.Fatalf("non-allowlisted command should be blocked: %#v", steps[0])
	}
}

func TestExecutePlanCommandsRecordsFailureEvidence(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test command uses POSIX sh")
	}
	dir := t.TempDir()
	writeFile(t, dir, "plan.json", `{
  "commands": [
    {
      "id": "cmd-1",
      "command": "sh -c 'exit 7'",
      "working_dir": ".",
      "timeout_seconds": 5
    }
  ]
}`)

	evidence, err := ExecutePlanCommands(dir, filepath.Join(dir, "plan.json"), []string{"sh"})
	if err != nil {
		t.Fatalf("command failure should be represented as evidence, got error: %v", err)
	}
	steps := evidence["steps"].([]map[string]any)
	if evidence["pass"] != false || steps[0]["status"] != "failed" || steps[0]["exit_code"] != 7 {
		t.Fatalf("unexpected failure evidence: %#v", evidence)
	}
}

func TestExecutePlanCommandsRecordsTimeoutEvidence(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test command uses POSIX sh")
	}
	dir := t.TempDir()
	writeFile(t, dir, "plan.json", `{
  "commands": [
    {
      "id": "cmd-1",
      "command": "sh -c 'sleep 2'",
      "working_dir": ".",
      "timeout_seconds": 1
    }
  ]
}`)

	evidence, err := ExecutePlanCommands(dir, filepath.Join(dir, "plan.json"), []string{"sh"})
	if err != nil {
		t.Fatalf("timeout should be represented as evidence, got error: %v", err)
	}
	steps := evidence["steps"].([]map[string]any)
	if evidence["pass"] != false || steps[0]["status"] != "timeout" || steps[0]["exit_code"] != 124 {
		t.Fatalf("unexpected timeout evidence: %#v", evidence)
	}
}

func TestExecutePlanCommandsRedactsSensitiveLogs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test command uses POSIX sh")
	}
	dir := t.TempDir()
	writeFile(t, dir, "plan.json", `{
  "commands": [
    {
      "id": "cmd-1",
      "command": "sh -c 'echo token=abc123 password=hunter2 secret:top dsn=postgres://user:pass@db/app'",
      "working_dir": ".",
      "timeout_seconds": 5
    }
  ]
}`)

	evidence, err := ExecutePlanCommands(dir, filepath.Join(dir, "plan.json"), []string{"sh"})
	if err != nil {
		t.Fatalf("ExecutePlanCommands returned error: %v", err)
	}
	logs := evidence["logs"].([]map[string]any)
	rendered := logs[0]["summary"].(string)
	for _, forbidden := range []string{"abc123", "hunter2", "top", "postgres://user:pass@db/app"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("sensitive value %q leaked in log summary: %s", forbidden, rendered)
		}
	}
	if !strings.Contains(rendered, "token=<redacted>") || !strings.Contains(rendered, "password=<redacted>") {
		t.Fatalf("expected redacted sensitive keys in log summary: %s", rendered)
	}
}

func writeFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}
