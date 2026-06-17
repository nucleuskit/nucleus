package report

import (
	"fmt"
	"path/filepath"
	"strings"
)

func run(config Config, opts *options) (reportResult, error) {
	if err := validateOptions(opts); err != nil {
		return reportResult{}, err
	}
	dir := stringValue(config.Dir, defaultDir)
	if opts.platform {
		return buildPlatformReadinessResult(dir), nil
	}
	tasksDir := strings.TrimSpace(opts.aiTasksPath)
	explicit := tasksDir != ""
	if !explicit {
		tasksDir = filepath.Join(dir, defaultAITasksDir)
	}
	return buildAIQualityResult(tasksDir, explicit), nil
}

func validateOptions(opts *options) error {
	if opts.platform && strings.TrimSpace(opts.aiTasksPath) != "" {
		return fmt.Errorf("--%s cannot be combined with --%s", flagPlatform, flagAITasks)
	}
	return nil
}

func stringValue(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}
