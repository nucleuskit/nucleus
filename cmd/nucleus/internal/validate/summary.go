package validate

import (
	"os"
	"path/filepath"

	"github.com/nucleuskit/contract/diagnostic"
)

const (
	inputManifest = "nucleus.yaml"
	inputOpenAPI  = "api/openapi.yaml"
	inputErrors   = "api/errors.yaml"
	inputProto    = "api/proto"
)

func buildSummary(dir string, diagnostics diagnostic.Diagnostics) validateSummary {
	summary := validateSummary{
		Errors:   diagnostics.Count(diagnostic.SeverityError),
		Warnings: diagnostics.Count(diagnostic.SeverityWarning),
	}
	addRequiredInput(dir, inputManifest, &summary)
	addOptionalInput(dir, inputOpenAPI, &summary)
	addOptionalInput(dir, inputErrors, &summary)
	addOptionalInput(dir, inputProto, &summary)
	return summary
}

func addRequiredInput(dir string, path string, summary *validateSummary) {
	if inputExists(dir, path) {
		summary.Checked = append(summary.Checked, path)
	}
}

func addOptionalInput(dir string, path string, summary *validateSummary) {
	if inputExists(dir, path) {
		summary.Checked = append(summary.Checked, path)
		return
	}
	summary.MissingOptional = append(summary.MissingOptional, path)
}

func inputExists(dir string, path string) bool {
	_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(path)))
	return err == nil
}
