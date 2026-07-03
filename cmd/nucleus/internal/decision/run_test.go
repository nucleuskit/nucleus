package decision

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateDecisionAcceptsStructuredEvidence(t *testing.T) {
	dir := t.TempDir()
	writeDecisionFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - id: order_store
    kind: relational_store
ai:
  intent: test
  allowed_changes:
    - nucleus.yaml
    - .nucleus/**
    - internal/**
`)
	writeDecisionFile(t, dir, ".nucleus/decisions/order-store.yaml", validDecisionYAML(""))

	output := validate(Config{Dir: &dir}, []string{".nucleus/decisions/order-store.yaml"})
	if !output.OK {
		t.Fatalf("ok = false, diagnostics = %#v", output.Diagnostics)
	}
	if output.Summary.Files != 1 || output.Summary.Valid != 1 {
		t.Fatalf("summary = %#v", output.Summary)
	}
	if output.Decisions[0].Hash == "" {
		t.Fatalf("decision hash missing: %#v", output.Decisions[0])
	}
}

func TestValidateDecisionReportsManifestAndSurfaceFailures(t *testing.T) {
	dir := t.TempDir()
	writeDecisionFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  name: demo
  version: "0.1.0"
capabilities: []
ai:
  intent: test
  allowed_changes:
    - nucleus.yaml
    - .nucleus/**
`)
	writeDecisionFile(t, dir, ".nucleus/decisions/order-store.yaml", validDecisionYAML(""))

	output := validate(Config{Dir: &dir}, []string{".nucleus/decisions/order-store.yaml"})
	if output.OK {
		t.Fatalf("ok = true, want failure")
	}
	assertDecisionDiagnostic(t, output, "decision.capability_missing")
	assertDecisionDiagnostic(t, output, "decision.impact_file_outside_edit_surface")
}

func TestValidateDecisionChecksLockedHash(t *testing.T) {
	dir := t.TempDir()
	writeDecisionFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
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
	writeDecisionFile(t, dir, ".nucleus/decisions/order-store.yaml", validDecisionYAML("sha256:bad"))

	output := validate(Config{Dir: &dir}, []string{".nucleus/decisions/order-store.yaml"})
	if output.OK {
		t.Fatalf("ok = true, want failure")
	}
	assertDecisionDiagnostic(t, output, "decision.hash_mismatch")
}

func TestValidateDecisionChecksSupersedesHash(t *testing.T) {
	dir := t.TempDir()
	writeDecisionFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
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
	writeDecisionFile(t, dir, ".nucleus/decisions/original.yaml", validDecisionYAML(""))
	writeDecisionFile(t, dir, ".nucleus/decisions/supersede.yaml", `schema_version: "decision.v1"
id: order-store-provider-v2
capability: order_store
supersedes: order-store-provider
supersedes_hash: sha256:not-the-original
decision:
  provider: database/sql
  library: database/sql
  status: proposed
  locked: false
reason:
  - replace gorm with standard library database sql
impact:
  files:
    - internal/order/store.go
verification:
  commands:
    - go test ./internal/order
`)

	output := validate(Config{Dir: &dir}, []string{".nucleus/decisions"})
	if output.OK {
		t.Fatalf("ok = true, want failure")
	}
	assertDecisionDiagnostic(t, output, "decision.supersedes_hash_mismatch")
}

func TestValidateDecisionAllowsUnknownProviderChoices(t *testing.T) {
	dir := t.TempDir()
	writeDecisionFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - id: order_store
    kind: project_specific_store_kind
ai:
  intent: test
  allowed_changes:
    - internal/**
`)
	writeDecisionFile(t, dir, ".nucleus/decisions/custom-provider.yaml", `schema_version: "decision.v1"
id: custom-provider
capability: order_store
decision:
  provider: private-infra-store
  library: example.com/private/store
  driver: team-owned-driver
  status: proposed
  locked: false
reason:
  - project owns the integration details
impact:
  files:
    - internal/order/store.go
verification:
  commands:
    - go test ./internal/order
alternatives:
  - provider: database/sql
    reason: standard library option
`)

	output := validate(Config{Dir: &dir}, []string{".nucleus/decisions/custom-provider.yaml"})
	if !output.OK {
		t.Fatalf("ok = false, diagnostics = %#v", output.Diagnostics)
	}
}

func TestAcceptDecisionLocksAndWritesHash(t *testing.T) {
	dir := t.TempDir()
	writeDecisionFile(t, dir, "nucleus.yaml", decisionManifestYAML())
	writeDecisionFile(t, dir, ".nucleus/decisions/order-store.yaml", proposedDecisionYAML("order-store-provider", "gorm", "gorm.io/gorm", ""))

	output := accept(Config{Dir: &dir}, ".nucleus/decisions/order-store.yaml", "human", "2026-07-03T00:00:00Z")
	if !output.OK {
		t.Fatalf("accept failed: %#v", output.Diagnostics)
	}
	if !output.Changed || output.Decision.Hash == "" {
		t.Fatalf("accept output = %#v", output)
	}
	doc, diagnostics, ok := loadDecisionFile(filepath.Join(dir, ".nucleus/decisions/order-store.yaml"), ".nucleus/decisions/order-store.yaml")
	if !ok || diagnostics.Failed() {
		t.Fatalf("load accepted decision: ok=%v diagnostics=%#v", ok, diagnostics)
	}
	if doc.Decision.Status != decisionStatusAccepted || doc.Decision.Locked == nil || !*doc.Decision.Locked {
		t.Fatalf("decision not accepted+locked: %#v", doc.Decision)
	}
	if doc.Decision.AcceptedBy != "human" || doc.Decision.AcceptedAt != "2026-07-03T00:00:00Z" {
		t.Fatalf("accepted metadata = %#v", doc.Decision)
	}
	if doc.DecisionHash == "" || doc.DecisionHash != canonicalDecisionHash(doc) {
		t.Fatalf("decision hash = %q, want canonical %q", doc.DecisionHash, canonicalDecisionHash(doc))
	}
}

func TestQualityForDirSummarizesLockedDecisionsAndDrift(t *testing.T) {
	dir := t.TempDir()
	writeDecisionFile(t, dir, "nucleus.yaml", decisionManifestYAML())
	writeDecisionFile(t, dir, ".nucleus/decisions/order-store.yaml", proposedDecisionYAML("order-store-provider", "gorm", "gorm.io/gorm", ""))

	accepted := accept(Config{Dir: &dir}, ".nucleus/decisions/order-store.yaml", "human", "2026-07-03T00:00:00Z")
	if !accepted.OK {
		t.Fatalf("accept failed: %#v", accepted.Diagnostics)
	}
	quality := QualityForDir(dir)
	if quality.Files != 1 || quality.Valid != 1 || quality.AcceptedLocked != 1 || quality.Drift != 0 {
		t.Fatalf("quality = %#v, want one valid locked decision without drift", quality)
	}

	writeDecisionFile(t, dir, ".nucleus/decisions/order-store.yaml", validDecisionYAML("sha256:bad"))
	quality = QualityForDir(dir)
	if quality.Drift != 1 || quality.Errors == 0 {
		t.Fatalf("quality = %#v, want drift error", quality)
	}
}

func TestQualityForDirIgnoresMissingDecisionDirectory(t *testing.T) {
	quality := QualityForDir(t.TempDir())
	if quality.Files != 0 || quality.Errors != 0 || quality.Warnings != 0 || len(quality.Diagnostics) != 0 {
		t.Fatalf("quality = %#v, want empty optional decision summary", quality)
	}
}

func TestSupersedeDecisionWritesPreviousHash(t *testing.T) {
	dir := t.TempDir()
	writeDecisionFile(t, dir, "nucleus.yaml", decisionManifestYAML())
	writeDecisionFile(t, dir, ".nucleus/decisions/original.yaml", proposedDecisionYAML("order-store-provider", "gorm", "gorm.io/gorm", ""))
	accepted := accept(Config{Dir: &dir}, ".nucleus/decisions/original.yaml", "human", "2026-07-03T00:00:00Z")
	if !accepted.OK {
		t.Fatalf("accept original: %#v", accepted.Diagnostics)
	}
	writeDecisionFile(t, dir, ".nucleus/decisions/supersede.yaml", proposedDecisionYAML("order-store-provider-v2", "xorm", "xorm.io/xorm", "order-store-provider"))

	output := supersede(Config{Dir: &dir}, ".nucleus/decisions/supersede.yaml")
	if !output.OK {
		t.Fatalf("supersede failed: %#v", output.Diagnostics)
	}
	doc, diagnostics, ok := loadDecisionFile(filepath.Join(dir, ".nucleus/decisions/supersede.yaml"), ".nucleus/decisions/supersede.yaml")
	if !ok || diagnostics.Failed() {
		t.Fatalf("load supersede decision: ok=%v diagnostics=%#v", ok, diagnostics)
	}
	if doc.SupersedesHash != accepted.Decision.Hash {
		t.Fatalf("supersedes_hash = %q, want %q", doc.SupersedesHash, accepted.Decision.Hash)
	}
	validation := validate(Config{Dir: &dir}, []string{".nucleus/decisions"})
	if !validation.OK {
		t.Fatalf("validate supersede flow: %#v", validation.Diagnostics)
	}
}

func TestCommandRendersJSONBeforeReturningValidationError(t *testing.T) {
	dir := t.TempDir()
	writeDecisionFile(t, dir, "nucleus.yaml", `schema_version: "2.0"
service:
  name: demo
  version: "0.1.0"
capabilities: []
ai:
  intent: test
`)
	writeDecisionFile(t, dir, ".nucleus/decisions/bad.yaml", `schema_version: "decision.v1"
id: bad
capability: missing
decision:
  status: proposed
  locked: false
reason:
  - missing provider evidence
verification:
  commands:
    - go test ./...
`)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"validate", ".nucleus/decisions/bad.yaml", "--json", "--pretty"})

	err := cmd.Execute()
	if !errors.Is(err, ErrDecisionInvalid) {
		t.Fatalf("execute error = %v, want ErrDecisionInvalid", err)
	}
	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout.String())
	}
	if output["result_kind"] != resultKindDecision || output["ok"] != false {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func decisionManifestYAML() string {
	return `schema_version: "2.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - id: order_store
    kind: sql
ai:
  intent: test
  allowed_changes:
    - internal/**
    - .nucleus/**
`
}

func proposedDecisionYAML(id string, provider string, library string, supersedes string) string {
	supersedesLine := ""
	if supersedes != "" {
		supersedesLine = "supersedes: " + supersedes + "\n"
	}
	return `schema_version: "decision.v1"
id: ` + id + `
capability: order_store
` + supersedesLine + `decision:
  provider: ` + provider + `
  library: ` + library + `
  status: proposed
  locked: false
reason:
  - project needs explicit provider evidence
impact:
  files:
    - internal/order/store.go
verification:
  commands:
    - go test ./internal/order
`
}

func validDecisionYAML(hash string) string {
	locked := "false"
	hashLine := ""
	if hash != "" {
		locked = "true"
		hashLine = "decision_hash: " + hash + "\n"
	}
	return `schema_version: "decision.v1"
id: order-store-provider
capability: order_store
decision:
  provider: gorm
  library: gorm.io/gorm
  status: accepted
  locked: ` + locked + `
  accepted_by: human
  accepted_at: "2026-07-03T00:00:00Z"
` + hashLine + `reason:
  - project already uses gorm in storage package
impact:
  symbols:
    - OrderStore
  files:
    - internal/order/store.go
verification:
  commands:
    - go test ./internal/order
alternatives:
  - provider: database/sql
    reason: lower dependency footprint
`
}

func writeDecisionFile(t *testing.T, dir string, name string, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", name, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func assertDecisionDiagnostic(t *testing.T, output result, code string) {
	t.Helper()
	for _, item := range output.Diagnostics {
		if item.Code == code {
			return
		}
	}
	t.Fatalf("diagnostic %q not found in %#v", code, output.Diagnostics)
}
