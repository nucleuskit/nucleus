package report

import (
	"path/filepath"
	"strings"
)

func run(config Config, opts *options) (reportResult, error) {
	dir := stringValue(config.Dir, defaultDir)
	tasksDir := strings.TrimSpace(opts.aiTasksPath)
	explicit := tasksDir != ""
	if !explicit {
		tasksDir = filepath.Join(dir, defaultAITasksDir)
	}
	return buildAIQualityResult(dir, tasksDir, explicit), nil
}

// BuildForMCP returns the same structured report result used by the CLI.
func BuildForMCP(dir string, aiTasksPath string) any {
	opts := &options{aiTasksPath: aiTasksPath}
	result, _ := run(Config{Dir: &dir}, opts)
	return result
}

func stringValue(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}
