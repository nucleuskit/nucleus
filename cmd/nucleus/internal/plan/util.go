package plan

import "strings"

func anyStringSlice(value any) []string {
	switch items := value.(type) {
	case []string:
		return items
	case []any:
		result := make([]string, 0, len(items))
		for _, item := range items {
			text, ok := item.(string)
			if ok {
				result = append(result, text)
			}
		}
		return result
	default:
		return nil
	}
}

func blockedEditsCount(output map[string]any) int {
	switch items := output["blocked_edits"].(type) {
	case []string:
		return len(items)
	case []map[string]any:
		return len(items)
	case []any:
		return len(items)
	default:
		if summary, ok := output["summary"].(planSummary); ok {
			return summary.BlockedEdits
		}
		return 0
	}
}

func containsAny(text string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(strings.ToLower(text), strings.ToLower(value)) {
			return true
		}
	}
	return false
}

func hasString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
