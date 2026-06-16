package execute

import (
	"fmt"
	"strings"
)

func run(config Config, opts *options) (map[string]any, error) {
	planPath := strings.TrimSpace(opts.planPath)
	if planPath == "" {
		return nil, fmt.Errorf("--%s is required", flagPlan)
	}
	if len(opts.allowCommands) == 0 {
		return nil, fmt.Errorf("--%s is required", flagAllowCommand)
	}
	return ExecutePlanCommands(stringValue(config.Dir, defaultDir), planPath, opts.allowCommands)
}

func stringValue(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}
