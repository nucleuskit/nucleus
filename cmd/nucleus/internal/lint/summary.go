package lint

import (
	"sort"

	contractlint "github.com/nucleuskit/contract/lint"
)

type lintSummary struct {
	Findings    int      `json:"findings"`
	Rules       []string `json:"rules"`
	Checked     []string `json:"checked"`
	ActiveRules []string `json:"active_rules"`
	Mode        string   `json:"mode"`
}

func buildSummary(strict bool, findings []contractlint.Finding) lintSummary {
	summary := lintSummary{
		Findings:    len(findings),
		Rules:       []string{},
		Checked:     lintCheckedScopes(strict),
		ActiveRules: lintActiveRules(strict),
		Mode:        lintMode(strict),
	}
	if len(findings) == 0 {
		return summary
	}

	rules := map[string]bool{}
	for _, finding := range findings {
		if finding.Rule != "" {
			rules[finding.Rule] = true
		}
	}
	for rule := range rules {
		summary.Rules = append(summary.Rules, rule)
	}
	sort.Strings(summary.Rules)
	return summary
}

func lintMode(strict bool) string {
	if strict {
		return "strict"
	}
	return "default"
}

func lintCheckedScopes(strict bool) []string {
	scopes := []string{
		"nucleus.yaml",
		"core imports",
		"runtime imports",
		"top-level directories",
	}
	if !strict {
		return scopes
	}
	return append(scopes,
		"api/openapi.yaml",
		"api/errors.yaml",
		"internal/domain imports",
		"capability graph",
		"dependencies",
		"api/proto",
		"critical legacy imports",
		"generated freshness",
		"contract/schema",
	)
}

func lintActiveRules(strict bool) []string {
	rules := []string{"L006", "L008", "L009", "L011"}
	if !strict {
		return rules
	}
	return append(rules, "L001", "L002", "L003", "L004", "L005", "L013", "L007", "L010", "L012")
}
