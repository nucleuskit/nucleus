package repair

import (
	"fmt"
	"strings"
)

func run(config Config, opts *options) (map[string]any, error) {
	evidencePath := strings.TrimSpace(opts.evidencePath)
	if evidencePath == "" {
		return nil, fmt.Errorf("--%s is required", flagFromEvidence)
	}
	if opts.maxRounds <= 0 {
		return nil, fmt.Errorf("--%s must be greater than 0", flagMaxRounds)
	}
	return BuildEvidence(stringValue(config.Dir, defaultDir), evidencePath, opts.maxRounds)
}

func stringValue(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}
