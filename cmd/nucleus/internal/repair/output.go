package repair

import (
	"encoding/json"
	"fmt"
	"io"
)

func renderHuman(stdout io.Writer, stderr io.Writer, evidence map[string]any) {
	if evidencePass(evidence) {
		_, _ = fmt.Fprintln(stdout, "OK")
	} else {
		_, _ = fmt.Fprintln(stderr, "FAILED")
	}
	if status := stringField(evidence, "status"); status != "" {
		_, _ = fmt.Fprintf(stdout, "status: %s\n", status)
	}
	_, _ = fmt.Fprintf(stdout, "rounds: %d\n", valueLen(evidence["rounds"]))
	if verificationPass, ok := evidence["verification_pass"].(bool); ok {
		_, _ = fmt.Fprintf(stdout, "verification_pass: %t\n", verificationPass)
	}
}

func renderJSON(writer io.Writer, evidence map[string]any, pretty bool) error {
	encoder := json.NewEncoder(writer)
	if pretty {
		encoder.SetIndent(jsonIndentPrefix, jsonIndentValue)
	}
	return encoder.Encode(evidence)
}

func evidencePass(evidence map[string]any) bool {
	ok, _ := evidence["ok"].(bool)
	return ok
}

func valueLen(value any) int {
	switch typed := value.(type) {
	case []any:
		return len(typed)
	case []map[string]any:
		return len(typed)
	case []string:
		return len(typed)
	default:
		return 0
	}
}
