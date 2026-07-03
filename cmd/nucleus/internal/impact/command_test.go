package impact

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestImpactSymbolReturnsFilesAndTests(t *testing.T) {
	dir := t.TempDir()
	writeImpactFile(t, dir, "go.mod", "module example.com/orders\n\ngo 1.26.3\n")
	writeImpactFile(t, dir, "order/service.go", `package order

func CreateOrder() { validateOrder() }
func validateOrder() {}
func Caller() { CreateOrder() }
`)
	writeImpactFile(t, dir, "order/service_test.go", `package order

func TestCreateOrder(t *testing.T) {}
`)

	output := impactSymbol(Config{Dir: &dir}, "CreateOrder")
	if !output.OK {
		t.Fatalf("ok=false diagnostics=%#v", output.Diagnostics)
	}
	if len(output.AffectedFiles) != 2 {
		t.Fatalf("affected files = %#v", output.AffectedFiles)
	}
	if len(output.AffectedTests) != 1 || output.AffectedTests[0].Name != "TestCreateOrder" {
		t.Fatalf("affected tests = %#v", output.AffectedTests)
	}
	if len(output.Edges) == 0 {
		t.Fatal("impact edges missing")
	}
}

func TestImpactFileExpandsDeclaredSymbols(t *testing.T) {
	dir := t.TempDir()
	writeImpactFile(t, dir, "go.mod", "module example.com/orders\n\ngo 1.26.3\n")
	writeImpactFile(t, dir, "order/service.go", `package order

func CreateOrder() {}
func Caller() { CreateOrder() }
`)

	output := impactFile(Config{Dir: &dir}, "order/service.go")
	if !output.OK {
		t.Fatalf("ok=false diagnostics=%#v", output.Diagnostics)
	}
	if len(output.AffectedSymbols) < 3 {
		t.Fatalf("affected symbols = %#v", output.AffectedSymbols)
	}
	assertImpactFile(t, output, "order/service.go")
}

func TestImpactContractReturnsRoutes(t *testing.T) {
	dir := t.TempDir()
	writeImpactFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /orders:
    post:
      operationId: createOrder
      responses:
        "200":
          description: ok
`)
	writeImpactFile(t, dir, "api/errors.yaml", `errors:
  - code: 0
    message: ok
    http_status: 200
`)

	output := impactContract(Config{Dir: &dir}, "api/openapi.yaml")
	if !output.OK {
		t.Fatalf("ok=false diagnostics=%#v", output.Diagnostics)
	}
	if len(output.AffectedRoutes) != 1 {
		t.Fatalf("routes = %#v", output.AffectedRoutes)
	}
	assertImpactFile(t, output, "api/openapi.yaml")
}

func TestImpactCommandRendersJSONBeforeFailure(t *testing.T) {
	dir := t.TempDir()
	writeImpactFile(t, dir, "go.mod", "module example.com/empty\n\ngo 1.26.3\n")
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"symbol", "Missing", "--json"})

	err := cmd.Execute()
	if !errors.Is(err, ErrImpactFailed) {
		t.Fatalf("execute error = %v, want ErrImpactFailed", err)
	}
	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout.String())
	}
	if output["ok"] != false || output["result_kind"] != resultKindImpact {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func writeImpactFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertImpactFile(t *testing.T, output result, want string) {
	t.Helper()
	for _, file := range output.AffectedFiles {
		if file == want {
			return
		}
	}
	t.Fatalf("file %q not found in %#v", want, output.AffectedFiles)
}
