package repair

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
)

func TestCommandJSONManualActionReturnsSentinel(t *testing.T) {
	dir := t.TempDir()
	evidencePath := filepath.Join(dir, "evidence.json")
	writeFile(t, dir, "evidence.json", `{
  "result_kind": "nucleus.apply_evidence",
  "ok": false,
  "steps": [
    {
      "id": "custom_failure",
      "kind": "custom_failure",
      "status": "failed",
      "ok": false
    }
  ]
}`)

	cmd := NewCommand(Config{Dir: &dir})
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--json", "--from-evidence", evidencePath})

	err := cmd.Execute()
	if !errors.Is(err, ErrRepairFailed) {
		t.Fatalf("execute repair error = %v, want ErrRepairFailed", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty for JSON output", stderr.String())
	}

	var output struct {
		ResultKind string `json:"result_kind"`
		OK         bool   `json:"ok"`
		Status     string `json:"status"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if output.ResultKind != resultKindRepairEvidence || output.OK || output.Status != "needs_manual_action" {
		t.Fatalf("unexpected repair evidence: %#v", output)
	}
}
