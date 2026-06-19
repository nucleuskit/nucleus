package initcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
	contractgen "github.com/nucleuskit/contract/gen"
)

var (
	serviceNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	modulePathPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9._~-]*(/[a-z0-9][a-z0-9._~-]*)+$`)
)

func run(config Config, opts *options) (initResult, error) {
	normalized, err := normalizeOptions(opts)
	if err != nil {
		return failedResult(opts, optionDiagnostic(err)), err
	}
	dir := stringValue(config.Dir, defaultDir)
	if err := ensureEmptyDir(dir); err != nil {
		wrapped := fmt.Errorf("%w: %v", ErrInitFailed, err)
		return failedResult(normalized, errorDiagnostic("init.target_unavailable", err.Error())), wrapped
	}

	files, generated, err := templateFiles(normalized.template, normalized.name, normalized.module)
	if err != nil {
		wrapped := fmt.Errorf("%w: %v", ErrInitFailed, err)
		return failedResult(normalized, errorDiagnostic("init.template_failed", err.Error())), wrapped
	}
	agentFiles, err := initAgentTemplateFiles(normalized.template, normalized.agent)
	if err != nil {
		wrapped := fmt.Errorf("%w: %v", ErrInitFailed, err)
		return failedResult(normalized, errorDiagnostic("init.agent_failed", err.Error())), wrapped
	}
	for name, data := range agentFiles {
		files[name] = data
	}

	written := make([]string, 0, len(files))
	for _, name := range sortedMapKeys(files) {
		if err := writeTemplateFile(filepath.Join(dir, filepath.FromSlash(name)), files[name]); err != nil {
			wrapped := fmt.Errorf("%w: %v", ErrInitFailed, err)
			return failedResult(normalized, errorDiagnostic("init.write_failed", err.Error())), wrapped
		}
		written = append(written, name)
	}

	if normalized.template == templateService {
		result, err := contractgen.GenerateWithOptions(dir, contractgen.Options{HTTP: true, Errors: true})
		if err != nil {
			wrapped := fmt.Errorf("%w: %v", ErrInitFailed, err)
			return failedResult(normalized, errorDiagnostic("init.generate_failed", err.Error())), wrapped
		}
		for _, file := range result.Files {
			written = append(written, relativeFile(dir, file))
		}
	}

	written = appendUniqueStrings(nil, written...)
	sort.Strings(written)
	sort.Strings(generated)
	return initResult{
		ResultKind:  resultKindInit,
		OK:          true,
		Template:    normalized.template,
		ServiceName: normalized.name,
		Module:      normalized.module,
		Summary: initSummary{
			Files:     len(written),
			Generated: len(generated),
		},
		Files:       written,
		Generated:   generated,
		Diagnostics: diagnostic.Diagnostics{},
	}, nil
}

func normalizeOptions(opts *options) (*options, error) {
	normalized := *opts
	normalized.name = strings.TrimSpace(normalized.name)
	normalized.module = strings.TrimSpace(normalized.module)
	normalized.template = strings.TrimSpace(normalized.template)
	normalized.agent = strings.TrimSpace(normalized.agent)
	if normalized.template == "" {
		normalized.template = defaultTemplate
	}
	if normalized.name == "" {
		return nil, fmt.Errorf("--%s is required", flagName)
	}
	if !serviceNamePattern.MatchString(normalized.name) {
		return nil, fmt.Errorf("service name must match %s", serviceNamePattern.String())
	}
	if normalized.module == "" {
		return nil, fmt.Errorf("--%s is required", flagModule)
	}
	if invalidModulePath(normalized.module) {
		return nil, fmt.Errorf("module path must be a slash-separated import path")
	}
	switch normalized.template {
	case templateService, templateWorker, templateLibrary:
	default:
		return nil, fmt.Errorf("unknown template %q", normalized.template)
	}
	switch normalized.agent {
	case "", agentCodex:
	default:
		return nil, fmt.Errorf("unknown agent %q", normalized.agent)
	}
	return &normalized, nil
}

func invalidModulePath(module string) bool {
	if !modulePathPattern.MatchString(module) {
		return true
	}
	first, _, _ := strings.Cut(module, "/")
	if !strings.Contains(first, ".") {
		return true
	}
	return false
}

func failedResult(opts *options, diagnostics diagnostic.Diagnostics) initResult {
	if opts == nil {
		return initResult{ResultKind: resultKindInit, OK: false, Summary: summaryForDiagnostics(diagnostics), Files: []string{}, Diagnostics: diagnostics}
	}
	return initResult{
		ResultKind:  resultKindInit,
		OK:          false,
		Template:    opts.template,
		ServiceName: opts.name,
		Module:      opts.module,
		Summary:     summaryForDiagnostics(diagnostics),
		Files:       []string{},
		Diagnostics: diagnostics,
	}
}

func optionDiagnostic(err error) diagnostic.Diagnostics {
	message := err.Error()
	code := "init.invalid_options"
	switch {
	case strings.Contains(message, "--"+flagName):
		code = "init.name_required"
	case strings.Contains(message, "service name"):
		code = "init.name_invalid"
	case strings.Contains(message, "--"+flagModule):
		code = "init.module_required"
	case strings.Contains(message, "module path"):
		code = "init.module_invalid"
	case strings.Contains(message, "unknown template"):
		code = "init.template_unknown"
	case strings.Contains(message, "unknown agent"):
		code = "init.agent_unknown"
	}
	return errorDiagnostic(code, message)
}

func errorDiagnostic(code string, message string) diagnostic.Diagnostics {
	return diagnostic.Diagnostics{{
		Severity: diagnostic.SeverityError,
		Code:     code,
		Message:  message,
	}}
}

func summaryForDiagnostics(diagnostics diagnostic.Diagnostics) initSummary {
	return initSummary{
		Errors:   diagnostics.Count(diagnostic.SeverityError),
		Warnings: diagnostics.Count(diagnostic.SeverityWarning),
	}
}

func ensureEmptyDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return os.MkdirAll(dir, 0o755)
	}
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("target directory is not empty: %s", dir)
	}
	return nil
}

func writeTemplateFile(path string, data string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(data), 0o644)
}

func sortedMapKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func relativeFile(dir string, path string) string {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func appendUniqueStrings(values []string, next ...string) []string {
	seen := map[string]struct{}{}
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, value := range next {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		values = append(values, value)
		seen[value] = struct{}{}
	}
	return values
}
