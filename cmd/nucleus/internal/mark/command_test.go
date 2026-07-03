package mark

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nucleuskit/contract/manifest"
)

func TestMarkContractUpdatesManifestOnly(t *testing.T) {
	dir := t.TempDir()
	writeMarkFile(t, dir, manifestFileName, baseManifest())
	writeMarkFile(t, dir, "go.mod", "module example.com/mark\n\ngo 1.26.3\n")

	output := markContract(Config{Dir: &dir}, "http", &options{kind: "openapi", path: "api/openapi.yaml"})
	if !output.OK {
		t.Fatalf("ok=false diagnostics=%#v", output.Diagnostics)
	}
	if !output.Changed {
		t.Fatal("changed=false, want true")
	}
	m := loadMarkedManifest(t, dir)
	if len(m.Contracts) != 1 || m.Contracts[0].ID != "http" || m.Contracts[0].Kind != "openapi" || m.Contracts[0].Path != "api/openapi.yaml" {
		t.Fatalf("contracts = %#v", m.Contracts)
	}
	if _, err := os.Stat(filepath.Join(dir, "go.sum")); err == nil {
		t.Fatal("mark must not create go.sum")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat go.sum: %v", err)
	}
}

func TestMarkCapabilityResolvesAndDeclaresSymbols(t *testing.T) {
	dir := t.TempDir()
	writeMarkFile(t, dir, manifestFileName, baseManifest())
	writeMarkFile(t, dir, "go.mod", "module example.com/mark\n\ngo 1.26.3\n")
	writeMarkFile(t, dir, "order/store.go", "package order\ntype OrderStore interface { Save() error }\n")
	beforeGoMod := readMarkFile(t, dir, "go.mod")

	output := markCapability(Config{Dir: &dir}, "order_store", &options{
		kind:    "project_specific_store_kind",
		intent:  "persist orders",
		symbols: []string{"OrderStore", "FutureStore"},
	})
	if !output.OK {
		t.Fatalf("ok=false diagnostics=%#v", output.Diagnostics)
	}
	if !output.Changed {
		t.Fatal("changed=false, want true")
	}
	assertMarkDiagnostic(t, output, "mark.symbol_declared")

	m := loadMarkedManifest(t, dir)
	if len(m.Capabilities) != 1 {
		t.Fatalf("capabilities = %#v", m.Capabilities)
	}
	capability := m.Capabilities[0]
	if capability.ID != "order_store" || capability.Kind != "project_specific_store_kind" || capability.Intent != "persist orders" {
		t.Fatalf("capability = %#v", capability)
	}
	if len(capability.Symbols) != 2 {
		t.Fatalf("symbols = %#v", capability.Symbols)
	}
	assertSymbolRef(t, capability.Symbols, manifest.SymbolRef{ID: "go://example.com/mark/order#OrderStore", Status: statusResolved})
	assertSymbolRef(t, capability.Symbols, manifest.SymbolRef{Name: "FutureStore", Status: statusDeclared})
	if got := readMarkFile(t, dir, "go.mod"); got != beforeGoMod {
		t.Fatalf("go.mod was modified:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "go.sum")); err == nil {
		t.Fatal("mark capability must not create go.sum")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat go.sum: %v", err)
	}
	manifestText := readMarkFile(t, dir, manifestFileName)
	for _, forbidden := range []string{"provider:", "providers:", "library:", "driver:"} {
		if strings.Contains(manifestText, forbidden) {
			t.Fatalf("manifest leaked provider decision field %q:\n%s", forbidden, manifestText)
		}
	}
}

func TestMarkCapabilityFailsOnAmbiguousShortSymbol(t *testing.T) {
	dir := t.TempDir()
	writeMarkFile(t, dir, manifestFileName, baseManifest())
	writeMarkFile(t, dir, "go.mod", "module example.com/ambiguous\n\ngo 1.26.3\n")
	writeMarkFile(t, dir, "a/a.go", "package a\ntype Store interface{}\n")
	writeMarkFile(t, dir, "b/b.go", "package b\ntype Store interface{}\n")

	output := markCapability(Config{Dir: &dir}, "store", &options{kind: "relational_store", symbols: []string{"Store"}})
	if output.OK {
		t.Fatal("ok=true, want ambiguous failure")
	}
	if len(output.Candidates) != 2 {
		t.Fatalf("candidates = %#v", output.Candidates)
	}
	assertMarkDiagnostic(t, output, "mark.symbol_ambiguous")

	m := loadMarkedManifest(t, dir)
	if len(m.Capabilities) != 0 {
		t.Fatalf("capability should not be written on ambiguity: %#v", m.Capabilities)
	}
}

func TestMarkVerifyAppendsIdempotently(t *testing.T) {
	dir := t.TempDir()
	writeMarkFile(t, dir, manifestFileName, baseManifest())

	first := markVerify(Config{Dir: &dir}, "go test ./...")
	if !first.OK || !first.Changed {
		t.Fatalf("first output = %#v", first)
	}
	second := markVerify(Config{Dir: &dir}, "go test ./...")
	if !second.OK || second.Changed {
		t.Fatalf("second output = %#v", second)
	}
	m := loadMarkedManifest(t, dir)
	if got := strings.Join(m.Verify.Commands, ","); got != "go test ./..." {
		t.Fatalf("verify commands = %q", got)
	}
}

func TestMarkCommandRendersJSONBeforeFailure(t *testing.T) {
	dir := t.TempDir()
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"verify", "go test ./...", "--json"})

	err := cmd.Execute()
	if !errors.Is(err, ErrMarkFailed) {
		t.Fatalf("execute error = %v, want ErrMarkFailed", err)
	}
	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout.String())
	}
	if output["ok"] != false || output["result_kind"] != resultKindMark {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func baseManifest() string {
	return `schema_version: "2.0"
service:
  name: mark-demo
  version: "0.1.0"
ai:
  intent: test mark
`
}

func writeMarkFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readMarkFile(t *testing.T, dir string, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

func loadMarkedManifest(t *testing.T, dir string) manifest.Manifest {
	t.Helper()
	m, err := manifest.Load(dir)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	return m
}

func assertMarkDiagnostic(t *testing.T, output result, code string) {
	t.Helper()
	for _, item := range output.Diagnostics {
		if item.Code == code {
			return
		}
	}
	t.Fatalf("diagnostic %q not found in %#v", code, output.Diagnostics)
}

func assertSymbolRef(t *testing.T, symbols []manifest.SymbolRef, want manifest.SymbolRef) {
	t.Helper()
	for _, symbol := range symbols {
		if symbol == want {
			return
		}
	}
	t.Fatalf("symbol ref %#v not found in %#v", want, symbols)
}
