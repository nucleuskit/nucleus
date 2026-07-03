package root

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestDecisionValidateFromRoot(t *testing.T) {
	dir := t.TempDir()
	writeRootFixtureFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - id: order_store
    kind: relational_store
ai:
  intent: test
  allowed_changes:
    - internal/**
`)
	writeRootFixtureFile(t, dir, ".nucleus/decisions/order-store.yaml", `schema_version: "decision.v1"
id: order-store-provider
capability: order_store
decision:
  provider: gorm
  library: gorm.io/gorm
  status: proposed
  locked: false
reason:
  - project already uses gorm
impact:
  files:
    - internal/order/store.go
verification:
  commands:
    - go test ./internal/order
`)

	cmd := New()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--dir", dir, "decision", "validate", ".nucleus/decisions/order-store.yaml", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute decision validate: %v\n%s", err, stdout.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode decision output: %v\n%s", err, stdout.String())
	}
	assertString(t, output, "result_kind", "nucleus.decision_validate_result")
	assertBool(t, output, "ok", true)
}
