package scenario

import (
	"encoding/json"
	"fmt"
	"io"
)

func renderHuman(stdout io.Writer, stderr io.Writer, result any) {
	if isEvidence(result) {
		renderEvidenceHuman(stdout, stderr, result)
		return
	}
	switch typed := result.(type) {
	case map[string]any:
		_, _ = fmt.Fprintln(stdout, "OK")
		if kind, _ := typed["kind"].(string); kind != "" {
			_, _ = fmt.Fprintf(stdout, "kind: %s\n", kind)
		}
		if scenarios := valueLen(typed["scenarios"]); scenarios > 0 {
			_, _ = fmt.Fprintf(stdout, "scenarios: %d\n", scenarios)
		}
		if cases := valueLen(typed["cases"]); cases > 0 {
			_, _ = fmt.Fprintf(stdout, "cases: %d\n", cases)
		}
	case []HTTPCase:
		_, _ = fmt.Fprintln(stdout, "OK")
		_, _ = fmt.Fprintf(stdout, "cases: %d\n", len(typed))
	default:
		_, _ = fmt.Fprintln(stdout, "OK")
	}
}

func renderEvidenceHuman(stdout io.Writer, stderr io.Writer, result any) {
	if evidencePass(result) {
		_, _ = fmt.Fprintln(stdout, "OK")
	} else {
		_, _ = fmt.Fprintln(stderr, "FAILED")
	}
	evidence, _ := result.(map[string]any)
	if status, _ := evidence["status"].(string); status != "" {
		_, _ = fmt.Fprintf(stdout, "status: %s\n", status)
	}
	_, _ = fmt.Fprintf(stdout, "steps: %d\n", valueLen(evidence["steps"]))
	_, _ = fmt.Fprintf(stdout, "http_samples: %d\n", valueLen(evidence["http_samples"]))
	_, _ = fmt.Fprintf(stdout, "assertions: %d\n", valueLen(evidence["assertion_results"]))
}

func renderJSON(writer io.Writer, result any, pretty bool) error {
	encoder := json.NewEncoder(writer)
	if pretty {
		encoder.SetIndent(jsonIndentPrefix, jsonIndentValue)
	}
	return encoder.Encode(result)
}

func isEvidence(result any) bool {
	evidence, ok := result.(map[string]any)
	if !ok {
		return false
	}
	kind, _ := evidence["result_kind"].(string)
	return kind == httpEvidenceKind
}

func evidencePass(result any) bool {
	evidence, ok := result.(map[string]any)
	if !ok {
		return true
	}
	okField, ok := evidence["ok"].(bool)
	return !ok || okField
}

func valueLen(value any) int {
	switch typed := value.(type) {
	case []any:
		return len(typed)
	case []map[string]any:
		return len(typed)
	case []HTTPCase:
		return len(typed)
	case []string:
		return len(typed)
	default:
		return 0
	}
}
