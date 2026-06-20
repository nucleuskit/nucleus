package migrate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func writeReport(serviceDir string, reportPath string, result migrateResult) error {
	path, err := resolveReportPath(serviceDir, reportPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var buffer strings.Builder
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent(jsonIndentPrefix, jsonIndentValue)
	if err := encoder.Encode(finalizeResult(result)); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(buffer.String()), 0o644)
}

func resolveReportPath(serviceDir string, reportPath string) (string, error) {
	reportPath = strings.TrimSpace(reportPath)
	serviceAbs, err := filepath.Abs(serviceDir)
	if err != nil {
		return "", err
	}
	var target string
	if filepath.IsAbs(reportPath) {
		target = filepath.Clean(reportPath)
	} else {
		cleaned := filepath.Clean(reportPath)
		if cleaned == "." {
			return "", fmt.Errorf("--%s must resolve inside the service directory", flagReport)
		}
		target = filepath.Join(serviceAbs, cleaned)
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(serviceAbs, targetAbs)
	if err != nil || !isContainedRelativePath(rel) {
		return "", fmt.Errorf("--%s must resolve inside the service directory", flagReport)
	}
	return targetAbs, nil
}

func safeMigratePath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		return filepath.ToSlash(clean)
	}
	if cwd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(cwd, clean); err == nil && isContainedRelativePath(rel) {
			return filepath.ToSlash(rel)
		}
	}
	base := filepath.Base(clean)
	parent := filepath.Base(filepath.Dir(clean))
	if parent != "." && parent != string(filepath.Separator) && parent != "" {
		return filepath.ToSlash(filepath.Join("<external>", parent, base))
	}
	return filepath.ToSlash(filepath.Join("<external>", base))
}

func isContainedRelativePath(path string) bool {
	if path == "." {
		return true
	}
	if path == ".." {
		return false
	}
	return !strings.HasPrefix(path, ".."+string(filepath.Separator))
}

func safeErrorMessage(err error) string {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Op + " failed: " + pathErr.Err.Error()
	}
	return err.Error()
}
