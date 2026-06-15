package verify

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	sensitiveKeyPattern     = regexp.MustCompile(`(?i)\b(token|secret|password|cookie|private_key|dsn)(\s*[:=]\s*)[^\s]+`)
	authorizationPattern    = regexp.MustCompile(`(?i)\b(authorization\s*:\s*bearer\s+)[^\s]+`)
	goModuleDirectivePrefix = "module "
)

func sanitizeCommandOutput(raw string, dir string) string {
	output := strings.TrimSpace(raw)
	output = redactKnownPaths(output, dir)
	output = redactLocalModulePath(output, dir)
	output = authorizationPattern.ReplaceAllString(output, "${1}"+redactedValue)
	output = sensitiveKeyPattern.ReplaceAllString(output, "${1}${2}"+redactedValue)
	return truncateCommandOutput(output)
}

func redactKnownPaths(output string, dir string) string {
	if dir != "" {
		if abs, err := filepath.Abs(dir); err == nil {
			cleaned := filepath.Clean(abs)
			if cleaned != string(filepath.Separator) {
				output = strings.ReplaceAll(output, cleaned, ".")
				output = strings.ReplaceAll(output, filepath.ToSlash(cleaned), ".")
			}
		}
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		cleaned := filepath.Clean(home)
		if cleaned != string(filepath.Separator) {
			output = strings.ReplaceAll(output, cleaned, "~")
			output = strings.ReplaceAll(output, filepath.ToSlash(cleaned), "~")
		}
	}
	return output
}

func redactLocalModulePath(output string, dir string) string {
	modulePath := readLocalModulePath(dir)
	if modulePath == "" {
		return output
	}
	return strings.ReplaceAll(output, modulePath, "<module>")
}

func readLocalModulePath(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, goModuleDirectivePrefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, goModuleDirectivePrefix))
		}
	}
	return ""
}

func truncateCommandOutput(output string) string {
	runes := []rune(output)
	if len(runes) <= maxCommandOutputRunes {
		return output
	}
	return string(runes[:maxCommandOutputRunes]) + "\n" + truncatedOutputNotice
}
