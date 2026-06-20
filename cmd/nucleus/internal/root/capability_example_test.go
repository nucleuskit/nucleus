package root

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestCapabilityCommandWithGlobalDir(t *testing.T) {
	repoRoot := repositoryRoot(t)
	exampleDir := filepath.Join(repoRoot, "example", "hello-http")

	cmd := New()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dir", exampleDir, "capability", "add", "redis", "--dry-run", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute capability: %v\nstderr=%s\nstdout=%s", err, stderr.String(), stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode capability output: %v\n%s", err, stdout.String())
	}
	assertString(t, output, "result_kind", "nucleus.capability_result")
	assertBool(t, output, "ok", true)
	assertString(t, output, "capability", "redis")
	assertString(t, output, "provider", "redis")
}
