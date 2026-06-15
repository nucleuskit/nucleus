package verify

import (
	"fmt"
	"strings"

	"github.com/nucleuskit/contract/inspect"
)

func runGeneratedFreshness(dir string) verifyStep {
	step := verifyStep{
		Phase:   phaseGeneratedFreshness,
		Command: commandGeneratedFreshness,
		OK:      true,
	}

	items, err := inspect.GeneratedFreshnessForDir(dir)
	if err != nil {
		step.OK = false
		step.Error = "load nucleus.yaml failed"
		return step
	}
	step.GeneratedFreshness = items
	if len(items) == 0 {
		step.Output = "no generated targets declared"
		return step
	}

	lines := make([]string, 0, len(items))
	for _, item := range items {
		if !item.Fresh {
			step.OK = false
			lines = append(lines, fmt.Sprintf("%s: %s", item.Target, item.Reason))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: fresh", item.Target))
	}
	step.Output = strings.Join(lines, "\n")
	return step
}
