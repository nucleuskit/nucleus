package serve

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCheckCommandRendersJSONEnvelope(t *testing.T) {
	dir := writeServiceFixture(t)
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--check", "--json", "--pretty"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute serve --check --json: %v\n%s", err, stdout.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if output["result_kind"] != resultKindServe {
		t.Fatalf("result_kind = %v, want %s", output["result_kind"], resultKindServe)
	}
	if output["schema_version"] != schemaVersionServe {
		t.Fatalf("schema_version = %v, want %s", output["schema_version"], schemaVersionServe)
	}
	if output["schema_ref"] != schemaRefServe {
		t.Fatalf("schema_ref = %v, want %s", output["schema_ref"], schemaRefServe)
	}
	if output["ok"] != true {
		t.Fatalf("ok = %v, want true", output["ok"])
	}
	if output["mode"] != modeCheck {
		t.Fatalf("mode = %v, want %s", output["mode"], modeCheck)
	}
	summary := requireMap(t, output, "summary")
	if summary["service"] != "demo" {
		t.Fatalf("summary.service = %v, want demo", summary["service"])
	}
	if summary["addr"] != defaultAddr {
		t.Fatalf("summary.addr = %v, want %s", summary["addr"], defaultAddr)
	}
	if summary["status"] != "ready" {
		t.Fatalf("summary.status = %v, want ready", summary["status"])
	}
	if summary["endpoint_count"] != float64(1) {
		t.Fatalf("summary.endpoint_count = %v, want 1", summary["endpoint_count"])
	}
	servedPaths := requireSlice(t, summary, "served_paths")
	if len(servedPaths) != 3 {
		t.Fatalf("len(summary.served_paths) = %d, want 3: %#v", len(servedPaths), servedPaths)
	}
	server := requireMap(t, output, "server")
	if server["listening"] != false {
		t.Fatalf("server.listening = %v, want false", server["listening"])
	}
	if server["network_scope"] != networkScopeLoopback {
		t.Fatalf("server.network_scope = %v, want %s", server["network_scope"], networkScopeLoopback)
	}
	metadataEndpoints := requireSlice(t, server, "metadata_endpoints")
	if len(metadataEndpoints) != 3 {
		t.Fatalf("len(server.metadata_endpoints) = %d, want 3", len(metadataEndpoints))
	}
	if !strings.Contains(stdout.String(), "\n  \"summary\"") {
		t.Fatalf("expected pretty JSON output, got %q", stdout.String())
	}
}

func TestCheckCommandRendersHumanSummary(t *testing.T) {
	dir := writeServiceFixture(t)
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--check"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute serve --check: %v\n%s", err, stdout.String())
	}

	output := stdout.String()
	for _, want := range []string{
		"OK\n",
		"mode: check\n",
		"service: demo\n",
		"addr: 127.0.0.1:8080\n",
		"served: /healthz, /readyz, /.well-known/nucleus.json\n",
		"metadata: endpoints=1 grpc_services=0 capabilities=1 generated_fresh=false\n",
		"diagnostics: 0 errors, 0 warnings\n",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("human output missing %q:\n%s", want, output)
		}
	}
}

func TestCheckCommandRejectsNonLocalAddrByDefault(t *testing.T) {
	dir := writeServiceFixture(t)
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--check", "--json", "--addr", "0.0.0.0:8080"})

	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected non-local addr to fail")
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	assertDiagnosticCode(t, output, diagnosticNonLocalAddr)
}

func TestCheckCommandAllowsNonLocalAddrWhenExplicit(t *testing.T) {
	dir := writeServiceFixture(t)
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--check", "--json", "--addr", "0.0.0.0:8080", "--allow-non-local"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute serve --check --allow-non-local: %v\n%s", err, stdout.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	server := requireMap(t, output, "server")
	if server["network_scope"] != networkScopeNonLocal {
		t.Fatalf("server.network_scope = %v, want %s", server["network_scope"], networkScopeNonLocal)
	}
}

func TestCheckCommandNormalizesEmptyAddrToDefault(t *testing.T) {
	dir := writeServiceFixture(t)
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--check", "--json", "--addr", ""})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute serve --check --addr empty: %v\n%s", err, stdout.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	summary := requireMap(t, output, "summary")
	if summary["addr"] != defaultAddr {
		t.Fatalf("summary.addr = %v, want %s", summary["addr"], defaultAddr)
	}
}

func TestCheckCommandReportsInvalidAddrScope(t *testing.T) {
	dir := writeServiceFixture(t)
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--check", "--json", "--addr", "not-an-address"})

	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected invalid addr to fail")
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	assertDiagnosticCode(t, output, diagnosticAddrInvalid)
	server := requireMap(t, output, "server")
	if server["network_scope"] != networkScopeInvalid {
		t.Fatalf("server.network_scope = %v, want %s", server["network_scope"], networkScopeInvalid)
	}
}

func TestServeCommandReportsActualBoundAddrAndShutsDown(t *testing.T) {
	dir := writeServiceFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := NewCommand(Config{Dir: &dir})
	var stdout lockedBuffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"--json", "--addr", " 127.0.0.1:0 "})

	errc := make(chan error, 1)
	go func() {
		errc <- cmd.Execute()
	}()

	output := waitForJSONOutput(t, &stdout)
	summary := requireMap(t, output, "summary")
	addr, ok := summary["addr"].(string)
	if !ok || addr == "" || strings.HasSuffix(addr, ":0") {
		t.Fatalf("summary.addr = %v, want actual bound address", summary["addr"])
	}
	server := requireMap(t, output, "server")
	if server["listening"] != true {
		t.Fatalf("server.listening = %v, want true", server["listening"])
	}

	response, err := http.Get("http://" + addr + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want 200", response.StatusCode)
	}

	cancel()
	select {
	case err := <-errc:
		if err != nil {
			t.Fatalf("serve returned error after cancel: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not shut down after context cancel")
	}
}

func TestServeCommandRendersJSONDiagnosticsOnListenFailure(t *testing.T) {
	dir := writeServiceFixture(t)
	listener, err := listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen test port: %v", err)
	}
	defer listener.Close()

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--json", "--addr", listener.Addr().String()})

	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected occupied addr to fail")
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	assertDiagnosticCode(t, output, diagnosticListenFailed)
	server := requireMap(t, output, "server")
	if server["listening"] != false {
		t.Fatalf("server.listening = %v, want false", server["listening"])
	}
}

type lockedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *lockedBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(data)
}

func (b *lockedBuffer) bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.buffer.Bytes()...)
}

func waitForJSONOutput(t *testing.T, buffer *lockedBuffer) map[string]any {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		data := buffer.bytes()
		if len(data) > 0 {
			var output map[string]any
			if err := json.Unmarshal(data, &output); err == nil {
				return output
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for JSON output; got %q", string(buffer.bytes()))
	return nil
}

func TestCheckCommandRendersJSONDiagnosticsOnInspectFailure(t *testing.T) {
	dir := t.TempDir()
	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--check", "--json"})

	if err := cmd.Execute(); err == nil {
		t.Fatalf("expected serve --check --json to fail for missing manifest")
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if output["ok"] != false {
		t.Fatalf("ok = %v, want false", output["ok"])
	}
	assertDiagnosticCode(t, output, diagnosticInspectFailed)
}

func assertDiagnosticCode(t *testing.T, output map[string]any, want string) {
	t.Helper()
	diagnostics := requireSlice(t, output, "diagnostics")
	if len(diagnostics) != 1 {
		t.Fatalf("len(diagnostics) = %d, want 1: %#v", len(diagnostics), diagnostics)
	}
	item, ok := diagnostics[0].(map[string]any)
	if !ok {
		t.Fatalf("diagnostic has type %T, want map[string]any", diagnostics[0])
	}
	if item["code"] != want {
		t.Fatalf("diagnostic.code = %v, want %s", item["code"], want)
	}
}

func requireMap(t *testing.T, value map[string]any, key string) map[string]any {
	t.Helper()
	item, ok := value[key].(map[string]any)
	if !ok {
		t.Fatalf("%s has type %T, want map[string]any", key, value[key])
	}
	return item
}

func requireSlice(t *testing.T, value map[string]any, key string) []any {
	t.Helper()
	item, ok := value[key].([]any)
	if !ok {
		t.Fatalf("%s has type %T, want []any", key, value[key])
	}
	return item
}
