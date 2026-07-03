package gen

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderJSONUsesStableEmptyArrays(t *testing.T) {
	var stdout bytes.Buffer

	if err := renderJSON(&stdout, genResult{
		OK:         true,
		SourceHash: "abc123",
		Summary:    genSummary{Files: 0},
	}, false); err != nil {
		t.Fatalf("renderJSON: %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	if got := output["result_kind"]; got != resultKindGen {
		t.Fatalf("result_kind = %v, want %s", got, resultKindGen)
	}
	if got := output["schema_version"]; got != schemaVersionGen {
		t.Fatalf("schema_version = %v, want %s", got, schemaVersionGen)
	}
	if got := output["schema_ref"]; got != schemaRefGen {
		t.Fatalf("schema_ref = %v, want %s", got, schemaRefGen)
	}
	if _, ok := output["files"].([]any); !ok {
		t.Fatalf("files has type %T, want []any", output["files"])
	}
	if _, ok := output["diagnostics"].([]any); !ok {
		t.Fatalf("diagnostics has type %T, want []any", output["diagnostics"])
	}
}

func TestRenderHumanValidationFailureUsesStderr(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	renderHuman(&stdout, &stderr, genResult{
		OK:      false,
		Summary: genSummary{Errors: 1},
	})

	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "FAILED") {
		t.Fatalf("stderr = %q, want FAILED", stderr.String())
	}
}
