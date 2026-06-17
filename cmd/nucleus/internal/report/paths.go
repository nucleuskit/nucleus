package report

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func safeReportPath(path string) string {
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
	if base == "ai-tasks" {
		return filepath.ToSlash(filepath.Join("<external>", base))
	}
	if parent == "ai-tasks" {
		return filepath.ToSlash(filepath.Join("<external>", parent, base))
	}
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
