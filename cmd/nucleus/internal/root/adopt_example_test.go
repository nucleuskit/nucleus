package root

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAdoptCommandFromRootCreatesProtocolIndex(t *testing.T) {
	dir := t.TempDir()
	writeRootFixtureFile(t, dir, "go.mod", "module example.com/adopted\n\ngo 1.26.3\n")

	cmd := New()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--dir", dir, "adopt", "--json", "--pretty"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute adopt: %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode adopt output: %v\n%s", err, stdout.String())
	}
	assertString(t, output, "result_kind", "nucleus.adopt_result")
	if output["ok"] != true {
		t.Fatalf("ok = %#v, want true; output=%s", output["ok"], stdout.String())
	}
	for _, name := range []string{"nucleus.yaml", ".nucleus/decisions/.gitkeep", ".nucleus/README.md"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(name))); err != nil {
			t.Fatalf("%s should exist: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "go.sum")); err == nil {
		t.Fatal("adopt must not create go.sum")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat go.sum: %v", err)
	}
}
