package root

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestMarkFromRoot(t *testing.T) {
	dir := t.TempDir()
	writeRootFixtureFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  name: mark-root
  version: "0.1.0"
ai:
  intent: root mark test
`)
	writeRootFixtureFile(t, dir, "go.mod", "module example.com/rootmark\n\ngo 1.26.3\n")
	writeRootFixtureFile(t, dir, "store/store.go", "package store\ntype OrderStore interface{}\n")

	output := executeRootJSON(t, dir, "mark", "capability", "order_store", "--kind", "relational_store", "--symbol", "OrderStore", "--json")
	assertString(t, output, "result_kind", "nucleus.mark_result")
	assertBool(t, output, "ok", true)
	assertBool(t, output, "changed", true)
	symbols := assertSlice(t, output, "symbols")
	if len(symbols) != 1 {
		t.Fatalf("symbols = %#v", symbols)
	}

	verifyOutput := executeRootJSON(t, dir, "mark", "verify", "go test ./...", "--json")
	assertString(t, verifyOutput, "result_kind", "nucleus.mark_result")
	assertBool(t, verifyOutput, "ok", true)
}

func TestMarkAmbiguousSymbolFromRootRendersJSON(t *testing.T) {
	dir := t.TempDir()
	writeRootFixtureFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  name: mark-root
  version: "0.1.0"
ai:
  intent: root mark test
`)
	writeRootFixtureFile(t, dir, "go.mod", "module example.com/rootmark\n\ngo 1.26.3\n")
	writeRootFixtureFile(t, dir, "a/a.go", "package a\ntype Store interface{}\n")
	writeRootFixtureFile(t, dir, "b/b.go", "package b\ntype Store interface{}\n")

	cmd := New()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--dir", dir, "mark", "capability", "store", "--kind", "relational_store", "--symbol", "Store", "--json"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("execute error = nil, want failure")
	}
	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout.String())
	}
	assertString(t, output, "result_kind", "nucleus.mark_result")
	assertBool(t, output, "ok", false)
	if got := assertSlice(t, output, "candidates"); len(got) != 2 {
		t.Fatalf("candidates = %#v", got)
	}
}
