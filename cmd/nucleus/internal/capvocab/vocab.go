// Package capvocab loads advisory capability kind vocabulary from data.
package capvocab

import (
	"embed"
	"encoding/json"
	"sort"
	"strings"
	"unicode"
)

//go:embed capability-kinds.v1.json
var files embed.FS

// Kind is an advisory capability kind entry.
type Kind struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Planning    bool     `json:"planning"`
	Aliases     []string `json:"aliases"`
}

type document struct {
	SchemaVersion string `json:"schema_version"`
	Kinds         []Kind `json:"kinds"`
}

// Kinds returns all advisory capability kinds from the embedded vocabulary data.
func Kinds() []Kind {
	doc := mustLoad()
	kinds := append([]Kind{}, doc.Kinds...)
	sort.Slice(kinds, func(i, j int) bool {
		return kinds[i].Name < kinds[j].Name
	})
	return kinds
}

// PlanningNames returns capability kinds considered by natural-language plans.
func PlanningNames() []string {
	var names []string
	for _, kind := range Kinds() {
		if kind.Planning {
			names = append(names, kind.Name)
		}
	}
	sort.Strings(names)
	return names
}

// MatchTask returns advisory capability kinds matched by name or aliases.
func MatchTask(task string) []string {
	lowerTask := strings.ToLower(task)
	var matches []string
	for _, kind := range Kinds() {
		if !kind.Planning {
			continue
		}
		for _, alias := range aliases(kind) {
			if aliasMatches(lowerTask, strings.ToLower(alias)) {
				matches = append(matches, kind.Name)
				break
			}
		}
	}
	return uniqueSorted(matches)
}

func mustLoad() document {
	data, err := files.ReadFile("capability-kinds.v1.json")
	if err != nil {
		panic(err)
	}
	var doc document
	if err := json.Unmarshal(data, &doc); err != nil {
		panic(err)
	}
	return doc
}

func aliases(kind Kind) []string {
	values := append([]string{kind.Name}, kind.Aliases...)
	return values
}

func aliasMatches(task string, alias string) bool {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return false
	}
	if containsNonASCII(alias) {
		return strings.Contains(task, alias)
	}
	if strings.Contains(alias, " ") || strings.Contains(alias, "-") {
		return strings.Contains(task, alias)
	}
	return containsASCIIToken(task, alias)
}

func containsASCIIToken(text string, token string) bool {
	for index := 0; ; {
		found := strings.Index(text[index:], token)
		if found < 0 {
			return false
		}
		start := index + found
		end := start + len(token)
		if asciiBoundary(text, start-1) && asciiBoundary(text, end) {
			return true
		}
		index = end
	}
}

func asciiBoundary(text string, index int) bool {
	if index < 0 || index >= len(text) {
		return true
	}
	r := rune(text[index])
	return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_')
}

func containsNonASCII(value string) bool {
	for _, r := range value {
		if r > unicode.MaxASCII {
			return true
		}
	}
	return false
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
