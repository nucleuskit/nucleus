package capvocab

import (
	"encoding/json"
	"testing"
)

func TestVocabularyHasNoProviderDecisions(t *testing.T) {
	data, err := files.ReadFile("capability-kinds.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	kinds, ok := raw["kinds"].([]any)
	if !ok || len(kinds) == 0 {
		t.Fatalf("kinds = %#v, want non-empty array", raw["kinds"])
	}
	for _, item := range kinds {
		kind, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("kind has type %T", item)
		}
		for _, forbidden := range []string{"provider", "providers", "default_provider", "driver", "library", "dsn_env"} {
			if _, exists := kind[forbidden]; exists {
				t.Fatalf("vocabulary leaked %s in %#v", forbidden, kind)
			}
		}
	}
}

func TestMatchTaskUsesVocabularyAliases(t *testing.T) {
	matches := MatchTask("add mysql capability")
	if !contains(matches, "sql") {
		t.Fatalf("matches = %#v, want sql", matches)
	}

	matches = MatchTask("move capability kind suggestions from Go catalog to vocab data")
	if contains(matches, "log") {
		t.Fatalf("catalog should not match log: %#v", matches)
	}

	matches = MatchTask("接入消息队列并增加指标")
	if !contains(matches, "mq") || !contains(matches, "metric") {
		t.Fatalf("matches = %#v, want mq and metric", matches)
	}
}

func TestPlanningNamesComeFromVocabulary(t *testing.T) {
	names := PlanningNames()
	if !contains(names, "sql") || !contains(names, "log") {
		t.Fatalf("planning names = %#v, want common capability kinds", names)
	}
	for _, forbidden := range []string{"zap", "gorm", "postgres", "kafka"} {
		if contains(names, forbidden) {
			t.Fatalf("planning names leaked provider %q: %#v", forbidden, names)
		}
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
