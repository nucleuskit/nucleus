package scenario

import (
	"fmt"
	"strings"
)

func run(config Config, opts *options) (any, error) {
	if err := validateOptions(opts); err != nil {
		return nil, err
	}
	dir := stringValue(config.Dir, defaultDir)
	switch {
	case strings.TrimSpace(opts.casesPath) != "":
		cases, err := LoadHTTPCases(opts.casesPath)
		if err != nil {
			return nil, err
		}
		return RunHTTPCases(HTTPRunnerOptions{BaseURL: opts.baseURL}, cases)
	case opts.draftCases:
		return buildHTTPCaseDraftOutput(dir)
	case opts.runHTTP:
		return RunHTTPScenarios(dir, HTTPRunnerOptions{BaseURL: opts.baseURL})
	default:
		return BuildScenarioPlan(dir)
	}
}

func validateOptions(opts *options) error {
	modes := 0
	if strings.TrimSpace(opts.casesPath) != "" {
		modes++
	}
	if opts.draftCases {
		modes++
	}
	if opts.runHTTP {
		modes++
	}
	if modes > 1 {
		return fmt.Errorf("--%s, --%s, and --%s are mutually exclusive", flagCases, flagDraftCases, flagRunHTTP)
	}
	if strings.TrimSpace(opts.baseURL) != "" && strings.TrimSpace(opts.casesPath) == "" && !opts.runHTTP {
		return fmt.Errorf("--%s requires --%s or --%s", flagBaseURL, flagRunHTTP, flagCases)
	}
	if strings.TrimSpace(opts.casesPath) != "" && strings.TrimSpace(opts.baseURL) == "" {
		return fmt.Errorf("--%s requires --%s", flagCases, flagBaseURL)
	}
	if opts.runHTTP && strings.TrimSpace(opts.baseURL) == "" {
		return fmt.Errorf("--%s requires --%s", flagRunHTTP, flagBaseURL)
	}
	return nil
}

func stringValue(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}
