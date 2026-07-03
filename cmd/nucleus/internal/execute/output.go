package execute

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
	_, _ = fmt.Fprintf(stdout, "steps: %d\n", valueLen(evidence["steps"]))
	if failed := failedSteps(evidence); failed > 0 {
		_, _ = fmt.Fprintf(stdout, "failed_steps: %d\n", failed)
	}
	_, _ = fmt.Fprintf(stdout, "exit_codes: %d\n", valueLen(evidence["exit_codes"]))
	_, _ = fmt.Fprintf(stdout, "assertions: %d\n", valueLen(evidence["assertion_results"]))
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

func stringField(evidence map[string]any, key string) string {
	value, _ := evidence[key].(string)
	return value
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

func failedSteps(evidence map[string]any) int {
	failed := 0
	switch steps := evidence["steps"].(type) {
	case []any:
		for _, value := range steps {
			step, ok := value.(map[string]any)
			if ok && step["ok"] == false {
				failed++
			}
		}
	case []map[string]any:
		for _, step := range steps {
			if step["ok"] == false {
				failed++
			}
		}
	}
	return failed
}
