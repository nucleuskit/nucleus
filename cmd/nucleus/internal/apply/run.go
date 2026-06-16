package apply

import (
	"fmt"
	"strings"
)

func run(config Config, opts *options) (map[string]any, error) {
	planPath := strings.TrimSpace(opts.planPath)
	if planPath == "" {
		return nil, fmt.Errorf("--%s is required", flagPlan)
	}
	dir := stringValue(config.Dir, defaultDir)
	if opts.dryRun {
		return BuildEvidence(dir, planPath)
	}
	return Apply(dir, planPath)
}

func stringValue(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}
