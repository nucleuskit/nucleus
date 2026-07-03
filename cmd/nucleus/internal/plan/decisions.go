package plan

import (
	"strings"

	"github.com/nucleuskit/contract/manifest"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/decision"
)

type lockedDecisionBlock struct {
	DecisionID     string   `json:"decision_id"`
	Path           string   `json:"path"`
	Capability     string   `json:"capability"`
	Provider       string   `json:"provider,omitempty"`
	Library        string   `json:"library,omitempty"`
	Driver         string   `json:"driver,omitempty"`
	Hash           string   `json:"hash"`
	Matched        []string `json:"matched"`
	Reason         string   `json:"reason"`
	RequiredAction string   `json:"required_action"`
}

func lockedDecisionBlocks(task string, requestedCapabilities []string, capabilities []manifest.Capability, state decision.PlanState) []lockedDecisionBlock {
	if len(state.Locked) == 0 || supersedeIntent(task) || !providerChangeIntent(task) {
		return []lockedDecisionBlock{}
	}
	var blocks []lockedDecisionBlock
	for _, locked := range state.Locked {
		if state.Supersedes[locked.ID] {
			continue
		}
		matches := lockedDecisionMatches(task, requestedCapabilities, capabilities, locked)
		if len(matches) == 0 {
			continue
		}
		blocks = append(blocks, lockedDecisionBlock{
			DecisionID:     locked.ID,
			Path:           locked.Path,
			Capability:     locked.Capability,
			Provider:       locked.Provider,
			Library:        locked.Library,
			Driver:         locked.Driver,
			Hash:           locked.Hash,
			Matched:        matches,
			Reason:         "task appears to change provider/library/driver protected by a locked decision",
			RequiredAction: "create and validate a supersede decision with supersedes and supersedes_hash before planning implementation changes",
		})
	}
	if blocks == nil {
		return []lockedDecisionBlock{}
	}
	return blocks
}

func lockedDecisionMatches(task string, requested []string, capabilities []manifest.Capability, locked decision.LockedChoice) []string {
	var matches []string
	text := normalizedDecisionTask(task)
	if containsDecisionToken(text, locked.Capability) {
		matches = append(matches, "capability:"+locked.Capability)
	}
	for _, value := range []string{locked.Provider, locked.Library, locked.Driver} {
		if containsDecisionToken(text, value) {
			matches = append(matches, "locked_choice:"+value)
		}
	}
	for _, capability := range capabilities {
		if capability.ID != locked.Capability {
			continue
		}
		for _, value := range []string{capability.ID, capability.Kind} {
			if containsDecisionToken(text, value) || hasString(requested, value) {
				matches = append(matches, "capability:"+value)
			}
		}
	}
	return uniqueStrings(matches)
}

func providerChangeIntent(task string) bool {
	return containsAny(task,
		"replace", "switch", "migrate", "change provider", "change library", "change driver",
		"use xorm", "use gorm", "use database/sql", "use mysql", "use postgres",
		"替换", "切换", "改用", "更换", "迁移", "换成", "使用 xorm", "使用 gorm",
	)
}

func supersedeIntent(task string) bool {
	return containsAny(task, "supersede", "supersedes", "supersede decision", "替代决策", "取代决策", "新 supersede")
}

func normalizedDecisionTask(task string) string {
	task = strings.ToLower(strings.ReplaceAll(task, "\\", "/"))
	replacer := strings.NewReplacer("-", " ", "_", " ", ".", " ", ",", " ", ":", " ", ";", " ", "\n", " ", "\t", " ")
	return " " + strings.Join(strings.Fields(replacer.Replace(task)), " ") + " "
}

func containsDecisionToken(text string, value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return false
	}
	normalized := normalizedDecisionTask(value)
	normalized = strings.TrimSpace(normalized)
	if normalized == "" {
		return false
	}
	return strings.Contains(text, " "+normalized+" ")
}

func blockedDecisionCount(output map[string]any) int {
	switch items := output["blocked_decisions"].(type) {
	case []lockedDecisionBlock:
		return len(items)
	case []map[string]any:
		return len(items)
	case []any:
		return len(items)
	default:
		if summary, ok := output["summary"].(planSummary); ok {
			return summary.BlockedDecisions
		}
		return 0
	}
}
