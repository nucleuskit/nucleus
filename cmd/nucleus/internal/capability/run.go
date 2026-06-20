package capability

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/capcatalog"
)

type fileOperation struct {
	path      string
	content   []byte
	generated bool
}

func run(config Config, opts *options, capabilityName string) (capabilityResult, error) {
	dir := stringValue(config.Dir, defaultDir)
	spec, provider, diagnostics := normalizeRequest(capabilityName, opts.provider)
	result := capabilityResult{
		ResultKind:    resultKindCapability,
		SchemaVersion: schemaVersion,
		SchemaRef:     schemaRefEvidence,
		OK:            !diagnostics.Failed(),
		Mode:          "add",
		Capability:    spec.Name,
		Provider:      provider,
		DryRun:        opts.dryRun,
		Forced:        opts.force,
		Diagnostics:   diagnostics,
	}
	if diagnostics.Failed() {
		result.Summary = summarize(nil, diagnostics)
		return result, fmt.Errorf("%w: invalid capability request", ErrCapabilityFailed)
	}

	operations, buildDiagnostics := buildOperations(dir, spec, provider)
	result.Diagnostics = append(result.Diagnostics, buildDiagnostics...)
	if result.Diagnostics.Failed() {
		result.OK = false
		result.Summary = summarize(nil, result.Diagnostics)
		return result, fmt.Errorf("%w: scaffold planning failed", ErrCapabilityFailed)
	}

	files, applyDiagnostics, err := applyOperations(dir, operations, opts)
	result.Files = files
	result.Diagnostics = append(result.Diagnostics, applyDiagnostics...)
	result.OK = !result.Diagnostics.Failed()
	result.Summary = summarize(result.Files, result.Diagnostics)
	if result.OK {
		result.NextSteps = nextSteps(spec, provider, opts.dryRun)
	}
	if err != nil {
		return result, err
	}
	if !result.OK {
		return result, fmt.Errorf("%w: scaffold has conflicts", ErrCapabilityFailed)
	}
	return result, nil
}

func normalizeRequest(capabilityName string, providerName string) (capcatalog.Spec, string, diagnostic.Diagnostics) {
	spec, ok := capcatalog.Lookup(capabilityName)
	if !ok {
		return capcatalog.Spec{}, "", errorDiagnostic("capability.unsupported", fmt.Sprintf("unsupported capability %q", strings.TrimSpace(capabilityName)))
	}
	if !spec.Planning {
		return spec, "", errorDiagnostic("capability.runtime_unsupported", fmt.Sprintf("runtime capability %q is managed by nucleus init/gen and cannot be scaffolded with capability add", spec.Name))
	}
	provider := capcatalog.Normalize(providerName)
	if provider == "" {
		provider = spec.DefaultProvider
	}
	if !capcatalog.ProviderSupported(spec, provider) {
		return spec, provider, errorDiagnostic("capability.provider_unsupported", fmt.Sprintf("unsupported %s provider %q; supported providers: %s", spec.Name, provider, strings.Join(spec.ProviderNames(), ", ")))
	}
	return spec, provider, nil
}

func buildOperations(dir string, spec capcatalog.Spec, provider string) ([]fileOperation, diagnostic.Diagnostics) {
	manifestPath := filepath.Join(dir, "nucleus.yaml")
	manifestContent, diagnostics := updatedManifest(manifestPath, spec, provider)
	if diagnostics.Failed() {
		return nil, diagnostics
	}
	module, err := modulePath(filepath.Join(dir, "go.mod"))
	if err != nil {
		return nil, errorDiagnostic("capability.module_unavailable", err.Error())
	}

	ops := []fileOperation{{
		path:    manifestPath,
		content: manifestContent,
	}}
	if spec.Name == "sql" && provider == "postgres" {
		sqlOps, diagnostics := postgresOperations(dir, module, spec, provider)
		if diagnostics.Failed() {
			return nil, diagnostics
		}
		ops = append(ops, sqlOps...)
		return ops, nil
	}
	ops = append(ops, genericOperations(dir, module, spec, provider)...)
	return ops, nil
}

func genericOperations(dir string, module string, spec capcatalog.Spec, provider string) []fileOperation {
	providerFile := providerFileName(provider)
	return []fileOperation{
		{
			path:      filepath.Join(dir, "internal", "component", spec.Name, providerFile+".go"),
			content:   []byte(genericComponentTemplate(spec, provider)),
			generated: true,
		},
		{
			path:      filepath.Join(dir, "internal", "app", "capabilities_"+spec.Name+".go"),
			content:   []byte(genericAppTemplate(module, spec, provider)),
			generated: true,
		},
		{
			path:      filepath.Join(dir, "docs", "capabilities", spec.Name+"-"+providerFile+".md"),
			content:   []byte(genericDocsTemplate(spec, provider)),
			generated: true,
		},
	}
}

func postgresOperations(dir string, module string, spec capcatalog.Spec, provider string) ([]fileOperation, diagnostic.Diagnostics) {
	goMod, err := updateGoModRequires(filepath.Join(dir, "go.mod"), []moduleRequirement{{Path: postgresDriverModule, Version: postgresDriverVersion}})
	if err != nil {
		return nil, errorDiagnostic("capability.gomod_failed", err.Error())
	}
	goSum, err := updateGoSumEntries(filepath.Join(dir, "go.sum"), postgresGoSumEntries())
	if err != nil {
		return nil, errorDiagnostic("capability.gosum_failed", err.Error())
	}
	providerFile := providerFileName(provider)
	return []fileOperation{
		{path: filepath.Join(dir, "go.mod"), content: goMod},
		{path: filepath.Join(dir, "go.sum"), content: goSum},
		{
			path:      filepath.Join(dir, "internal", "component", "sql", providerFile+".go"),
			content:   []byte(postgresComponentTemplate()),
			generated: true,
		},
		{
			path:      filepath.Join(dir, "internal", "app", "capabilities_sql.go"),
			content:   []byte(postgresAppTemplate(module)),
			generated: true,
		},
		{
			path:      filepath.Join(dir, "internal", "adapter", "store", "postgres", "repository.go"),
			content:   []byte(postgresRepositoryTemplate()),
			generated: true,
		},
		{
			path:      filepath.Join(dir, "deploy", "migrations", "000001_sql_postgres.sql"),
			content:   []byte(postgresMigrationTemplate()),
			generated: true,
		},
		{
			path:      filepath.Join(dir, "docs", "capabilities", spec.Name+"-"+providerFile+".md"),
			content:   []byte(postgresDocsTemplate(spec, provider)),
			generated: true,
		},
	}, nil
}

func applyOperations(dir string, operations []fileOperation, opts *options) ([]fileChange, diagnostic.Diagnostics, error) {
	sort.Slice(operations, func(i, j int) bool {
		return operations[i].path < operations[j].path
	})
	var files []fileChange
	var diagnostics diagnostic.Diagnostics
	for _, op := range operations {
		change, itemDiagnostics := classifyOperation(dir, op, opts)
		files = append(files, change)
		diagnostics = append(diagnostics, itemDiagnostics...)
	}
	if diagnostics.Failed() {
		return files, diagnostics, fmt.Errorf("%w: scaffold has file conflicts", ErrCapabilityFailed)
	}
	if opts.dryRun {
		return files, diagnostics, nil
	}
	for _, op := range operations {
		if err := writeOperation(op); err != nil {
			item := diagnostic.Diagnostic{
				Severity: diagnostic.SeverityError,
				Code:     "capability.write_failed",
				Path:     relativeFile(dir, op.path),
				Message:  "write file failed",
			}
			diagnostics = append(diagnostics, item)
			return files, diagnostics, fmt.Errorf("%w: write file failed", ErrCapabilityFailed)
		}
	}
	return files, diagnostics, nil
}

func classifyOperation(dir string, op fileOperation, opts *options) (fileChange, diagnostic.Diagnostics) {
	rel := relativeFile(dir, op.path)
	data, err := os.ReadFile(op.path)
	if err != nil {
		if os.IsNotExist(err) {
			if opts.dryRun {
				return fileChange{Path: rel, Action: actionWouldCreate}, nil
			}
			return fileChange{Path: rel, Action: actionCreated}, nil
		}
		return fileChange{Path: rel, Action: actionConflict}, errorDiagnosticAt("capability.read_failed", rel, "read file failed")
	}
	if string(data) == string(op.content) {
		return fileChange{Path: rel, Action: actionUnchanged}, nil
	}
	if op.generated && !opts.force {
		return fileChange{Path: rel, Action: actionConflict}, errorDiagnosticAt("capability.file_conflict", rel, "generated scaffold file already exists with different content; rerun with --force to overwrite")
	}
	if opts.dryRun {
		return fileChange{Path: rel, Action: actionWouldUpdate}, nil
	}
	return fileChange{Path: rel, Action: actionUpdated}, nil
}

func writeOperation(op fileOperation) error {
	if err := os.MkdirAll(filepath.Dir(op.path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(op.path, op.content, 0o644)
}

func nextSteps(spec capcatalog.Spec, provider string, dryRun bool) []string {
	if dryRun {
		return []string{"rerun without --dry-run to apply the scaffold"}
	}
	steps := []string{
		"run nucleus validate --dir .",
		"run nucleus lint --dir . --strict",
		"run nucleus verify --dir . --json",
	}
	if spec.Name == "sql" && provider == "postgres" {
		steps = append([]string{"set NUCLEUS_DATABASE_DSN outside committed config before enabling the database connection"}, steps...)
	}
	return steps
}

func modulePath(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("go.mod is required and must be readable")
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			return fields[1], nil
		}
	}
	return "", fmt.Errorf("go.mod module path is required")
}

func relativeFile(dir string, path string) string {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func errorDiagnostic(code string, message string) diagnostic.Diagnostics {
	return diagnostic.Diagnostics{{
		Severity: diagnostic.SeverityError,
		Code:     code,
		Message:  message,
	}}
}

func errorDiagnosticAt(code string, path string, message string) diagnostic.Diagnostics {
	return diagnostic.Diagnostics{{
		Severity: diagnostic.SeverityError,
		Code:     code,
		Path:     path,
		Message:  message,
	}}
}
