package mark

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
	"github.com/nucleuskit/contract/inspect"
	"github.com/nucleuskit/contract/manifest"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/graphquery"
	"go.yaml.in/yaml/v3"
)

func markContract(config Config, id string, opts *options) result {
	dir := stringValue(config.Dir, defaultDir)
	m, diagnostics := loadManifestForMark(dir)
	normalized := contractInput{
		id:   strings.TrimSpace(id),
		kind: strings.TrimSpace(opts.kind),
		path: normalizeSlashPath(opts.path),
	}
	diagnostics = append(diagnostics, validateContractInput(normalized)...)
	if diagnostics.Failed() {
		return normalizeResult(result{Action: "contract", Diagnostics: diagnostics})
	}

	entry := manifest.Contract{ID: normalized.id, Kind: normalized.kind, Path: normalized.path}
	changed := upsertContract(&m, entry)
	if changed {
		diagnostics = append(diagnostics, writeManifest(dir, m)...)
	}
	return normalizeResult(result{
		Action:      "contract",
		Changed:     changed && !diagnostics.Failed(),
		Entry:       entry,
		Diagnostics: diagnostics,
	})
}

func markCapability(config Config, id string, opts *options) result {
	dir := stringValue(config.Dir, defaultDir)
	m, diagnostics := loadManifestForMark(dir)
	normalized := capabilityInput{
		id:      strings.TrimSpace(id),
		kind:    strings.TrimSpace(opts.kind),
		intent:  strings.TrimSpace(opts.intent),
		symbols: trimStrings(opts.symbols),
	}
	diagnostics = append(diagnostics, validateCapabilityInput(normalized)...)
	if diagnostics.Failed() {
		return normalizeResult(result{Action: "capability", Diagnostics: diagnostics})
	}

	refs, marks, candidates, resolveDiagnostics := resolveSymbolRefs(dir, normalized.symbols)
	diagnostics = append(diagnostics, resolveDiagnostics...)
	if diagnostics.Failed() {
		return normalizeResult(result{
			Action:      "capability",
			Symbols:     marks,
			Candidates:  candidates,
			Diagnostics: diagnostics,
		})
	}

	entry := manifest.Capability{
		ID:      normalized.id,
		Kind:    normalized.kind,
		Intent:  normalized.intent,
		Symbols: refs,
	}
	changed := upsertCapability(&m, entry)
	if changed {
		diagnostics = append(diagnostics, writeManifest(dir, m)...)
	}
	return normalizeResult(result{
		Action:      "capability",
		Changed:     changed && !diagnostics.Failed(),
		Entry:       capabilityEntry(m.Capabilities, normalized.id),
		Symbols:     marks,
		Diagnostics: diagnostics,
	})
}

func markVerify(config Config, command string) result {
	dir := stringValue(config.Dir, defaultDir)
	m, diagnostics := loadManifestForMark(dir)
	command = strings.TrimSpace(command)
	if command == "" {
		diagnostics = append(diagnostics, errorDiagnostic("mark.verify_command_required", manifestFileName, "verify command is required"))
	}
	if diagnostics.Failed() {
		return normalizeResult(result{Action: "verify", Diagnostics: diagnostics})
	}
	changed := appendVerifyCommand(&m, command)
	if changed {
		diagnostics = append(diagnostics, writeManifest(dir, m)...)
	}
	return normalizeResult(result{
		Action:      "verify",
		Changed:     changed && !diagnostics.Failed(),
		Entry:       command,
		Diagnostics: diagnostics,
	})
}

type contractInput struct {
	id   string
	kind string
	path string
}

type capabilityInput struct {
	id      string
	kind    string
	intent  string
	symbols []string
}

func loadManifestForMark(dir string) (manifest.Manifest, diagnostic.Diagnostics) {
	m, err := manifest.Load(dir)
	if err == nil {
		return m, manifest.ValidateDiagnostics(m)
	}
	if errors.Is(err, os.ErrNotExist) {
		return manifest.Manifest{}, diagnostic.Diagnostics{errorDiagnostic("mark.manifest_missing", manifestFileName, "nucleus.yaml is required; run nucleus adopt first")}
	}
	return manifest.Manifest{}, diagnostic.Diagnostics{errorDiagnostic("mark.manifest_load_failed", manifestFileName, err.Error())}
}

func validateContractInput(input contractInput) diagnostic.Diagnostics {
	var diagnostics diagnostic.Diagnostics
	if input.id == "" {
		diagnostics = append(diagnostics, errorDiagnostic("mark.contract_id_required", manifestFileName, "contract id is required"))
	}
	if input.kind == "" {
		diagnostics = append(diagnostics, errorDiagnostic("mark.contract_kind_required", manifestFileName, "contract kind is required"))
	}
	if input.path == "" {
		diagnostics = append(diagnostics, errorDiagnostic("mark.contract_path_required", manifestFileName, "contract path is required"))
	} else if !safeRelativePath(input.path) {
		diagnostics = append(diagnostics, errorDiagnostic("mark.contract_path_invalid", input.path, "contract path must be relative and stay inside the service directory"))
	}
	return diagnostics
}

func validateCapabilityInput(input capabilityInput) diagnostic.Diagnostics {
	var diagnostics diagnostic.Diagnostics
	if input.id == "" {
		diagnostics = append(diagnostics, errorDiagnostic("mark.capability_id_required", manifestFileName, "capability id is required"))
	}
	if input.kind == "" {
		diagnostics = append(diagnostics, errorDiagnostic("mark.capability_kind_required", manifestFileName, "capability kind is required"))
	}
	return diagnostics
}

func resolveSymbolRefs(dir string, queries []string) ([]manifest.SymbolRef, []symbolMark, []inspect.SymbolNode, diagnostic.Diagnostics) {
	if len(queries) == 0 {
		return nil, nil, nil, nil
	}
	description, err := inspect.Describe(dir)
	if err != nil {
		return nil, nil, nil, diagnostic.Diagnostics{errorDiagnostic("mark.describe_failed", manifestFileName, err.Error())}
	}
	var refs []manifest.SymbolRef
	var marks []symbolMark
	var candidates []inspect.SymbolNode
	var diagnostics diagnostic.Diagnostics
	for _, query := range queries {
		resolved := graphquery.ResolveSymbol(description.SymbolGraph, query)
		if resolved.OK {
			refs = append(refs, manifest.SymbolRef{ID: resolved.Node.ID, Status: statusResolved})
			marks = append(marks, symbolMark{Query: query, ID: resolved.Node.ID, Name: resolved.Node.Name, Status: statusResolved})
			continue
		}
		if len(resolved.Candidates) > 0 {
			candidates = append(candidates, resolved.Candidates...)
			diagnostics = append(diagnostics, errorDiagnostic("mark.symbol_ambiguous", manifestFileName, "symbol matched multiple candidates; rerun with a stable symbol id"))
			continue
		}
		ref := declaredSymbolRef(query)
		refs = append(refs, ref)
		marks = append(marks, symbolMark{Query: query, ID: ref.ID, Name: ref.Name, Status: statusDeclared})
		diagnostics = append(diagnostics, warningDiagnostic("mark.symbol_declared", manifestFileName, "symbol was not found and was recorded as declared intent"))
	}
	sortCandidates(candidates)
	return refs, marks, candidates, diagnostics
}

func declaredSymbolRef(query string) manifest.SymbolRef {
	query = strings.TrimSpace(query)
	if strings.HasPrefix(query, "go://") {
		return manifest.SymbolRef{ID: query, Status: statusDeclared}
	}
	return manifest.SymbolRef{Name: query, Status: statusDeclared}
}

func upsertContract(m *manifest.Manifest, entry manifest.Contract) bool {
	for index, existing := range m.Contracts {
		if existing.ID != entry.ID {
			continue
		}
		if existing == entry {
			return false
		}
		m.Contracts[index] = entry
		return true
	}
	m.Contracts = append(m.Contracts, entry)
	sort.Slice(m.Contracts, func(i, j int) bool { return m.Contracts[i].ID < m.Contracts[j].ID })
	return true
}

func upsertCapability(m *manifest.Manifest, entry manifest.Capability) bool {
	for index, existing := range m.Capabilities {
		if existing.ID != entry.ID {
			continue
		}
		updated := existing
		updated.Kind = entry.Kind
		if entry.Intent != "" {
			updated.Intent = entry.Intent
		}
		updated.Symbols = mergeSymbolRefs(updated.Symbols, entry.Symbols)
		if capabilitiesEqual(existing, updated) {
			return false
		}
		m.Capabilities[index] = updated
		return true
	}
	m.Capabilities = append(m.Capabilities, entry)
	sort.Slice(m.Capabilities, func(i, j int) bool { return m.Capabilities[i].ID < m.Capabilities[j].ID })
	return true
}

func appendVerifyCommand(m *manifest.Manifest, command string) bool {
	for _, existing := range m.Verify.Commands {
		if existing == command {
			return false
		}
	}
	m.Verify.Commands = append(m.Verify.Commands, command)
	return true
}

func mergeSymbolRefs(existing []manifest.SymbolRef, additions []manifest.SymbolRef) []manifest.SymbolRef {
	result := append([]manifest.SymbolRef{}, existing...)
	seen := map[string]bool{}
	for _, item := range result {
		seen[symbolKey(item)] = true
	}
	for _, item := range additions {
		key := symbolKey(item)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return symbolKey(result[i]) < symbolKey(result[j]) })
	return result
}

func symbolKey(symbol manifest.SymbolRef) string {
	if symbol.ID != "" {
		return "id:" + symbol.ID
	}
	return "name:" + symbol.Name
}

func capabilityEntry(capabilities []manifest.Capability, id string) manifest.Capability {
	for _, capability := range capabilities {
		if capability.ID == id {
			return capability
		}
	}
	return manifest.Capability{}
}

func capabilitiesEqual(left manifest.Capability, right manifest.Capability) bool {
	if left.ID != right.ID || left.Kind != right.Kind || left.Intent != right.Intent || len(left.Symbols) != len(right.Symbols) {
		return false
	}
	for index := range left.Symbols {
		if left.Symbols[index] != right.Symbols[index] {
			return false
		}
	}
	return true
}

func writeManifest(dir string, m manifest.Manifest) diagnostic.Diagnostics {
	data, err := yaml.Marshal(m)
	if err != nil {
		return diagnostic.Diagnostics{errorDiagnostic("mark.manifest_encode_failed", manifestFileName, err.Error())}
	}
	path := filepath.Join(dir, manifestFileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return diagnostic.Diagnostics{errorDiagnostic("mark.manifest_write_failed", manifestFileName, err.Error())}
	}
	return nil
}

func normalizeSlashPath(value string) string {
	return strings.TrimPrefix(strings.ReplaceAll(strings.TrimSpace(value), "\\", "/"), "./")
}

func safeRelativePath(value string) bool {
	if value == "" || strings.HasPrefix(value, "/") {
		return false
	}
	clean := pathClean(value)
	return clean != "." && clean != ".." && !strings.HasPrefix(clean, "../") && !strings.Contains(clean, "/../")
}

func pathClean(value string) string {
	return filepath.ToSlash(filepath.Clean(strings.ReplaceAll(value, "\\", "/")))
}

func trimStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func sortCandidates(candidates []inspect.SymbolNode) {
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
}

func errorDiagnostic(code string, path string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{Severity: diagnostic.SeverityError, Code: code, Path: path, Message: message}
}

func warningDiagnostic(code string, path string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{Severity: diagnostic.SeverityWarning, Code: code, Path: path, Message: message}
}
