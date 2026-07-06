package trace

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestTraceSymbolReturnsCallersAndCallees(t *testing.T) {
	dir := t.TempDir()
	writeTraceFile(t, dir, "go.mod", "module example.com/orders\n\ngo 1.26.3\n")
	writeTraceFile(t, dir, "order/service.go", `package order

func CreateOrder() { validateOrder() }
func validateOrder() {}
func Caller() { CreateOrder() }
`)

	output := traceSymbol(Config{Dir: &dir}, "CreateOrder")
	if !output.OK {
		t.Fatalf("ok=false diagnostics=%#v", output.Diagnostics)
	}
	if output.Query.ResolvedID != "go://example.com/orders/order#CreateOrder" {
		t.Fatalf("resolved id = %q", output.Query.ResolvedID)
	}
	if len(output.Callers) != 1 || output.Callers[0].Node.Name != "Caller" {
		t.Fatalf("callers = %#v", output.Callers)
	}
	if len(output.Callees) != 1 || output.Callees[0].Node.Name != "validateOrder" {
		t.Fatalf("callees = %#v", output.Callees)
	}
	for _, edge := range output.Edges {
		if edge.Source == "" || edge.Confidence == "" {
			t.Fatalf("edge missing metadata: %#v", edge)
		}
	}
}

func TestTraceSymbolReportsAmbiguousShortName(t *testing.T) {
	dir := t.TempDir()
	writeTraceFile(t, dir, "go.mod", "module example.com/ambiguous\n\ngo 1.26.3\n")
	writeTraceFile(t, dir, "a/a.go", "package a\nfunc Run() {}\n")
	writeTraceFile(t, dir, "b/b.go", "package b\nfunc Run() {}\n")

	output := traceSymbol(Config{Dir: &dir}, "Run")
	if output.OK {
		t.Fatal("ok=true, want ambiguous failure")
	}
	if len(output.Candidates) != 2 {
		t.Fatalf("candidates = %#v", output.Candidates)
	}
	assertTraceDiagnostic(t, output, "graph.symbol_ambiguous")
}

func TestTraceRouteUsesFlowGraph(t *testing.T) {
	dir := t.TempDir()
	writeTraceFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /orders:
    post:
      operationId: createOrder
      responses:
        "200":
          description: ok
`)
	writeTraceFile(t, dir, "api/errors.yaml", `errors:
  - code: 0
    message: ok
    http_status: 200
`)

	output := traceRoute(Config{Dir: &dir}, "POST /orders")
	if !output.OK {
		t.Fatalf("ok=false diagnostics=%#v", output.Diagnostics)
	}
	if len(output.FlowNodes) == 0 || len(output.FlowEdges) == 0 {
		t.Fatalf("flow trace missing nodes or edges: %#v %#v", output.FlowNodes, output.FlowEdges)
	}
}

func TestTraceCapabilityUsesManifestSymbolAnchors(t *testing.T) {
	dir := t.TempDir()
	writeTraceFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  name: orders
  version: "0.1.0"
capabilities:
  - id: order_flow
    kind: workflow
    symbols:
      - id: go://example.com/orders/order#CreateOrder
        status: resolved
ai:
  intent: test
`)
	writeTraceFile(t, dir, "go.mod", "module example.com/orders\n\ngo 1.26.3\n")
	writeTraceFile(t, dir, "order/service.go", `package order

func CreateOrder() { validateOrder() }
func validateOrder() {}
func Caller() { CreateOrder() }
`)

	output := traceCapability(Config{Dir: &dir}, "order_flow")
	if !output.OK {
		t.Fatalf("ok=false diagnostics=%#v", output.Diagnostics)
	}
	if output.Query.ResolvedID != "order_flow" {
		t.Fatalf("resolved id = %q", output.Query.ResolvedID)
	}
	if len(output.Callers) != 1 || output.Callers[0].Node.Name != "Caller" {
		t.Fatalf("callers = %#v", output.Callers)
	}
	if len(output.Callees) != 1 || output.Callees[0].Node.Name != "validateOrder" {
		t.Fatalf("callees = %#v", output.Callees)
	}
}

func TestTraceCapabilityWarnsOnDeclaredMissingSymbol(t *testing.T) {
	dir := t.TempDir()
	writeTraceFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  name: orders
  version: "0.1.0"
capabilities:
  - id: future_store
    kind: relational_store
    symbols:
      - name: FutureStore
        status: declared
ai:
  intent: test
`)
	writeTraceFile(t, dir, "go.mod", "module example.com/orders\n\ngo 1.26.3\n")

	output := traceCapability(Config{Dir: &dir}, "future_store")
	if !output.OK {
		t.Fatalf("ok=false diagnostics=%#v", output.Diagnostics)
	}
	assertTraceDiagnostic(t, output, "trace.capability_symbol_unresolved")
	if len(output.Edges) != 0 {
		t.Fatalf("edges = %#v, want empty for declared missing symbol", output.Edges)
	}
}

func TestTraceCommandRendersJSONBeforeFailure(t *testing.T) {
	dir := t.TempDir()
	writeTraceFile(t, dir, "go.mod", "module example.com/empty\n\ngo 1.26.3\n")
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"symbol", "Missing", "--json"})

	err := cmd.Execute()
	if !errors.Is(err, ErrTraceFailed) {
		t.Fatalf("execute error = %v, want ErrTraceFailed", err)
	}
	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout.String())
	}
	if output["ok"] != false || output["result_kind"] != resultKindTrace {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func writeTraceFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertTraceDiagnostic(t *testing.T, output result, code string) {
	t.Helper()
	for _, item := range output.Diagnostics {
		if item.Code == code {
			return
		}
	}
	t.Fatalf("diagnostic %q not found in %#v", code, output.Diagnostics)
}
