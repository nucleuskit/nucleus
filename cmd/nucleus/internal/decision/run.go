package decision

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nucleuskit/contract/diagnostic"
	"github.com/nucleuskit/contract/inspect"
	"github.com/nucleuskit/contract/manifest"
	"go.yaml.in/yaml/v3"
)

type document struct {
	SchemaVersion  string        `yaml:"schema_version" json:"schema_version"`
	ID             string        `yaml:"id" json:"id"`
	Capability     string        `yaml:"capability" json:"capability"`
	Decision       choice        `yaml:"decision" json:"decision"`
	DecisionHash   string        `yaml:"decision_hash" json:"decision_hash,omitempty"`
	Supersedes     string        `yaml:"supersedes" json:"supersedes,omitempty"`
	SupersedesHash string        `yaml:"supersedes_hash" json:"supersedes_hash,omitempty"`
	Reason         []string      `yaml:"reason" json:"reason"`
	Impact         impact        `yaml:"impact" json:"impact,omitempty"`
	Verification   verification  `yaml:"verification" json:"verification"`
	Alternatives   []alternative `yaml:"alternatives" json:"alternatives,omitempty"`
}

type choice struct {
	Provider   string `yaml:"provider" json:"provider,omitempty"`
	Library    string `yaml:"library" json:"library,omitempty"`
	Driver     string `yaml:"driver" json:"driver,omitempty"`
	Status     string `yaml:"status" json:"status"`
	Locked     *bool  `yaml:"locked" json:"locked"`
	AcceptedBy string `yaml:"accepted_by" json:"accepted_by,omitempty"`
	AcceptedAt string `yaml:"accepted_at" json:"accepted_at,omitempty"`
}

type impact struct {
	Symbols []string `yaml:"symbols" json:"symbols,omitempty"`
	Files   []string `yaml:"files" json:"files,omitempty"`
}

type verification struct {
	Commands []string `yaml:"commands" json:"commands"`
}

type alternative struct {
	Provider string `yaml:"provider" json:"provider,omitempty"`
	Library  string `yaml:"library" json:"library,omitempty"`
	Driver   string `yaml:"driver" json:"driver,omitempty"`
	Reason   string `yaml:"reason" json:"reason,omitempty"`
}

type loadedDecision struct {
	path string
	doc  document
}

// LockedChoice is the plan-facing view of an accepted locked decision.
type LockedChoice struct {
	Path       string `json:"path"`
	ID         string `json:"id"`
	Capability string `json:"capability"`
	Provider   string `json:"provider,omitempty"`
	Library    string `json:"library,omitempty"`
	Driver     string `json:"driver,omitempty"`
	Hash       string `json:"hash"`
}

// PlanState contains decision facts that plan can use without mutating files.
type PlanState struct {
	Locked      []LockedChoice         `json:"locked"`
	Supersedes  map[string]bool        `json:"supersedes"`
	Diagnostics diagnostic.Diagnostics `json:"diagnostics"`
}

func validate(config Config, args []string) result {
	dir := stringValue(config.Dir, defaultDir)
	files, diagnostics := collectDecisionFiles(dir, args)
	decisions := make([]fileSummary, 0, len(files))
	loaded := make([]loadedDecision, 0, len(files))
	for _, file := range files {
		doc, parseDiagnostics, ok := loadDecisionFile(file.fullPath, file.relPath)
		diagnostics = append(diagnostics, parseDiagnostics...)
		if !ok {
			continue
		}
		loaded = append(loaded, loadedDecision{path: file.relPath, doc: doc})
		decisions = append(decisions, summarizeDecision(file.relPath, doc))
	}

	m, manifestDiagnostics, manifestLoaded := loadManifest(dir)
	diagnostics = append(diagnostics, manifestDiagnostics...)
	description, describeDiagnostics, descriptionLoaded := loadDescription(dir)
	diagnostics = append(diagnostics, describeDiagnostics...)

	byID := decisionsByID(loaded)
	for _, item := range loaded {
		diagnostics = append(diagnostics, validateDocument(item, m, manifestLoaded, description, descriptionLoaded, byID)...)
	}
	diagnostics.Sort()
	return normalizeResult(result{Decisions: decisions, Diagnostics: diagnostics})
}

func accept(config Config, path string, acceptedBy string, acceptedAt string) actionResult {
	dir := stringValue(config.Dir, defaultDir)
	rel, full, diagnostics := resolveSingleDecisionFile(dir, path)
	if diagnostics.Failed() {
		return normalizeActionResult(actionResult{Action: "accept", Path: rel, Diagnostics: diagnostics})
	}
	doc, loadDiagnostics, ok := loadDecisionFile(full, rel)
	diagnostics = append(diagnostics, loadDiagnostics...)
	if !ok {
		return normalizeActionResult(actionResult{Action: "accept", Path: rel, Diagnostics: diagnostics})
	}
	acceptedAtValue, timeDiagnostics := normalizeAcceptedAt(acceptedAt)
	diagnostics = append(diagnostics, timeDiagnostics...)
	if strings.TrimSpace(acceptedBy) == "" {
		diagnostics = append(diagnostics, errorDiagnostic("decision.accepted_by_required", rel, "--by must not be empty"))
	}
	if diagnostics.Failed() {
		return normalizeActionResult(actionResult{Action: "accept", Path: rel, Decision: summarizeDecision(rel, doc), Diagnostics: diagnostics})
	}
	locked := true
	doc.Decision.Status = decisionStatusAccepted
	doc.Decision.Locked = &locked
	doc.Decision.AcceptedBy = strings.TrimSpace(acceptedBy)
	doc.Decision.AcceptedAt = acceptedAtValue
	doc.DecisionHash = canonicalDecisionHash(doc)
	diagnostics = append(diagnostics, writeDecisionDocument(full, rel, doc)...)
	if diagnostics.Failed() {
		return normalizeActionResult(actionResult{Action: "accept", Path: rel, Decision: summarizeDecision(rel, doc), Diagnostics: diagnostics})
	}
	validation := validate(config, []string{rel})
	diagnostics = append(diagnostics, validation.Diagnostics...)
	diagnostics.Sort()
	return normalizeActionResult(actionResult{Action: "accept", Path: rel, Changed: !diagnostics.Failed(), Decision: summarizeDecision(rel, doc), Diagnostics: diagnostics})
}

func supersede(config Config, path string) actionResult {
	dir := stringValue(config.Dir, defaultDir)
	rel, full, diagnostics := resolveSingleDecisionFile(dir, path)
	if diagnostics.Failed() {
		return normalizeActionResult(actionResult{Action: "supersede", Path: rel, Diagnostics: diagnostics})
	}
	doc, loadDiagnostics, ok := loadDecisionFile(full, rel)
	diagnostics = append(diagnostics, loadDiagnostics...)
	if !ok {
		return normalizeActionResult(actionResult{Action: "supersede", Path: rel, Diagnostics: diagnostics})
	}
	if strings.TrimSpace(doc.Supersedes) == "" {
		diagnostics = append(diagnostics, errorDiagnostic("decision.supersedes_required", rel, "supersede command requires supersedes"))
		return normalizeActionResult(actionResult{Action: "supersede", Path: rel, Decision: summarizeDecision(rel, doc), Diagnostics: diagnostics})
	}
	previous, previousDiagnostics, found := loadDecisionByID(dir, doc.Supersedes)
	diagnostics = append(diagnostics, previousDiagnostics...)
	if !found {
		diagnostics = append(diagnostics, errorDiagnostic("decision.supersedes_not_found", rel, "superseded decision was not found in .nucleus/decisions"))
		return normalizeActionResult(actionResult{Action: "supersede", Path: rel, Decision: summarizeDecision(rel, doc), Diagnostics: diagnostics})
	}
	doc.SupersedesHash = decisionHash(previous.doc)
	diagnostics = append(diagnostics, writeDecisionDocument(full, rel, doc)...)
	if diagnostics.Failed() {
		return normalizeActionResult(actionResult{Action: "supersede", Path: rel, Decision: summarizeDecision(rel, doc), Diagnostics: diagnostics})
	}
	validation := validate(config, []string{defaultDecisionDir})
	diagnostics = append(diagnostics, validation.Diagnostics...)
	diagnostics.Sort()
	return normalizeActionResult(actionResult{Action: "supersede", Path: rel, Changed: !diagnostics.Failed(), Decision: summarizeDecision(rel, doc), Diagnostics: diagnostics})
}

// ValidateForMCP returns the same structured decision validation result used by the CLI.
func ValidateForMCP(dir string, args []string) any {
	return validate(Config{Dir: &dir}, args)
}

// QualityForDir returns decision health without requiring every project to have
// a decisions directory.
func QualityForDir(dir string) QualitySummary {
	files, diagnostics := collectDecisionFiles(dir, []string{defaultDecisionDir})
	if len(files) == 0 {
		diagnostics = suppressDecisionDirMissing(diagnostics)
		diagnostics.Sort()
		return normalizeQualitySummary(QualitySummary{Diagnostics: diagnostics})
	}
	output := validate(Config{Dir: &dir}, []string{defaultDecisionDir})
	summary := QualitySummary{
		Files:          output.Summary.Files,
		Valid:          output.Summary.Valid,
		Errors:         output.Summary.Errors,
		Warnings:       output.Summary.Warnings,
		AcceptedLocked: acceptedLockedDecisionCount(output.Decisions),
		Supersedes:     supersedingDecisionCount(dir),
		Drift:          decisionDriftCount(output.Diagnostics),
		Diagnostics:    output.Diagnostics,
	}
	return normalizeQualitySummary(summary)
}

// PlanStateForDir returns locked choices and valid supersede facts for plan.
func PlanStateForDir(dir string) PlanState {
	files, diagnostics := collectDecisionFiles(dir, []string{defaultDecisionDir})
	var loaded []loadedDecision
	for _, file := range files {
		doc, itemDiagnostics, ok := loadDecisionFile(file.fullPath, file.relPath)
		diagnostics = append(diagnostics, itemDiagnostics...)
		if ok {
			loaded = append(loaded, loadedDecision{path: file.relPath, doc: doc})
		}
	}
	m, manifestDiagnostics, manifestLoaded := loadManifest(dir)
	diagnostics = append(diagnostics, manifestDiagnostics...)
	description, describeDiagnostics, descriptionLoaded := loadDescription(dir)
	diagnostics = append(diagnostics, describeDiagnostics...)
	byID := decisionsByID(loaded)
	pathHasError := map[string]bool{}
	for _, item := range loaded {
		itemDiagnostics := validateDocument(item, m, manifestLoaded, description, descriptionLoaded, byID)
		for _, item := range itemDiagnostics {
			if item.Severity == diagnostic.SeverityError && item.Path != "" {
				pathHasError[item.Path] = true
			}
		}
		diagnostics = append(diagnostics, itemDiagnostics...)
	}
	state := PlanState{Supersedes: map[string]bool{}}
	for _, item := range loaded {
		if pathHasError[item.path] {
			continue
		}
		if strings.TrimSpace(item.doc.Supersedes) != "" {
			state.Supersedes[item.doc.Supersedes] = true
		}
		if isAcceptedLocked(item.doc) {
			state.Locked = append(state.Locked, LockedChoice{
				Path:       item.path,
				ID:         item.doc.ID,
				Capability: item.doc.Capability,
				Provider:   item.doc.Decision.Provider,
				Library:    item.doc.Decision.Library,
				Driver:     item.doc.Decision.Driver,
				Hash:       decisionHash(item.doc),
			})
		}
	}
	diagnostics.Sort()
	state.Diagnostics = suppressDecisionDirMissing(diagnostics)
	if state.Locked == nil {
		state.Locked = []LockedChoice{}
	}
	return state
}

type decisionFile struct {
	relPath  string
	fullPath string
}

func collectDecisionFiles(dir string, args []string) ([]decisionFile, diagnostic.Diagnostics) {
	if len(args) == 0 {
		args = []string{defaultDecisionDir}
	}
	var diagnostics diagnostic.Diagnostics
	seen := map[string]bool{}
	var files []decisionFile
	for _, arg := range args {
		rel, full, err := resolveDecisionPath(dir, arg)
		if err != nil {
			diagnostics = append(diagnostics, errorDiagnostic("decision.path_invalid", arg, err.Error()))
			continue
		}
		info, err := os.Stat(full)
		if err != nil {
			if os.IsNotExist(err) && rel == defaultDecisionDir {
				diagnostics = append(diagnostics, warningDiagnostic("decision.dir_missing", rel, "decision directory does not exist"))
				continue
			}
			diagnostics = append(diagnostics, errorDiagnostic("decision.path_read_failed", rel, safeError(err)))
			continue
		}
		if !info.IsDir() {
			if !isDecisionFile(rel) {
				diagnostics = append(diagnostics, errorDiagnostic("decision.path_unsupported", rel, "decision path must be a .yaml, .yml, or .json file"))
				continue
			}
			if !seen[rel] {
				seen[rel] = true
				files = append(files, decisionFile{relPath: rel, fullPath: full})
			}
			continue
		}
		walkDiagnostics := collectDecisionDir(full, rel, seen, &files)
		diagnostics = append(diagnostics, walkDiagnostics...)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].relPath < files[j].relPath })
	return files, diagnostics
}

func collectDecisionDir(root string, rootRel string, seen map[string]bool, files *[]decisionFile) diagnostic.Diagnostics {
	var diagnostics diagnostic.Diagnostics
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			diagnostics = append(diagnostics, warningDiagnostic("decision.path_read_failed", rootRel, safeError(err)))
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		rel := filepath.ToSlash(filepath.Join(rootRel, strings.TrimPrefix(path, root)))
		rel = strings.TrimPrefix(rel, "/")
		if !isDecisionFile(rel) {
			return nil
		}
		if seen[rel] {
			return nil
		}
		seen[rel] = true
		*files = append(*files, decisionFile{relPath: rel, fullPath: path})
		return nil
	})
	if err != nil {
		diagnostics = append(diagnostics, errorDiagnostic("decision.path_read_failed", rootRel, safeError(err)))
	}
	return diagnostics
}

func resolveDecisionPath(dir string, value string) (string, string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", fmt.Errorf("decision path must not be empty")
	}
	value = filepath.ToSlash(value)
	if filepath.IsAbs(value) || strings.HasPrefix(value, "/") {
		return "", "", fmt.Errorf("decision path must be relative to --dir")
	}
	clean := filepath.ToSlash(filepath.Clean(value))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", "", fmt.Errorf("decision path must stay inside --dir")
	}
	return clean, filepath.Join(dir, filepath.FromSlash(clean)), nil
}

func isDecisionFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml", ".json":
		return true
	default:
		return false
	}
}

func loadDecisionFile(path string, rel string) (document, diagnostic.Diagnostics, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return document{}, diagnostic.Diagnostics{errorDiagnostic("decision.read_failed", rel, safeError(err))}, false
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var doc document
	if err := decoder.Decode(&doc); err != nil {
		return document{}, diagnostic.Diagnostics{errorDiagnostic("decision.parse_failed", rel, safeError(err))}, false
	}
	return doc, nil, true
}

func summarizeDecision(path string, doc document) fileSummary {
	locked := false
	if doc.Decision.Locked != nil {
		locked = *doc.Decision.Locked
	}
	return fileSummary{
		Path:       path,
		ID:         doc.ID,
		Capability: doc.Capability,
		Status:     doc.Decision.Status,
		Locked:     locked,
		Hash:       decisionHash(doc),
	}
}

func decisionHash(doc document) string {
	if strings.TrimSpace(doc.DecisionHash) != "" {
		return doc.DecisionHash
	}
	return canonicalDecisionHash(doc)
}

func resolveSingleDecisionFile(dir string, value string) (string, string, diagnostic.Diagnostics) {
	rel, full, err := resolveDecisionPath(dir, value)
	if err != nil {
		return rel, full, diagnostic.Diagnostics{errorDiagnostic("decision.path_invalid", value, err.Error())}
	}
	info, err := os.Stat(full)
	if err != nil {
		return rel, full, diagnostic.Diagnostics{errorDiagnostic("decision.path_read_failed", rel, safeError(err))}
	}
	if info.IsDir() {
		return rel, full, diagnostic.Diagnostics{errorDiagnostic("decision.path_must_be_file", rel, "decision action requires one file")}
	}
	if !isDecisionFile(rel) {
		return rel, full, diagnostic.Diagnostics{errorDiagnostic("decision.path_unsupported", rel, "decision path must be a .yaml, .yml, or .json file")}
	}
	return rel, full, nil
}

func writeDecisionDocument(path string, rel string, doc document) diagnostic.Diagnostics {
	var data []byte
	var err error
	if strings.EqualFold(filepath.Ext(rel), ".json") {
		data, err = json.MarshalIndent(doc, "", "  ")
		if err == nil {
			data = append(data, '\n')
		}
	} else {
		data, err = yaml.Marshal(doc)
	}
	if err != nil {
		return diagnostic.Diagnostics{errorDiagnostic("decision.encode_failed", rel, safeError(err))}
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return diagnostic.Diagnostics{errorDiagnostic("decision.write_failed", rel, safeError(err))}
	}
	return nil
}

func normalizeAcceptedAt(value string) (string, diagnostic.Diagnostics) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Now().UTC().Format(time.RFC3339), nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return "", diagnostic.Diagnostics{errorDiagnostic("decision.accepted_at_invalid", "", "accepted-at must be RFC3339")}
	}
	return parsed.UTC().Format(time.RFC3339), nil
}

func loadDecisionByID(dir string, id string) (loadedDecision, diagnostic.Diagnostics, bool) {
	files, diagnostics := collectDecisionFiles(dir, []string{defaultDecisionDir})
	for _, file := range files {
		doc, itemDiagnostics, ok := loadDecisionFile(file.fullPath, file.relPath)
		diagnostics = append(diagnostics, itemDiagnostics...)
		if ok && doc.ID == id {
			return loadedDecision{path: file.relPath, doc: doc}, diagnostics, true
		}
	}
	return loadedDecision{}, diagnostics, false
}

func isAcceptedLocked(doc document) bool {
	return strings.TrimSpace(doc.Decision.Status) == decisionStatusAccepted && doc.Decision.Locked != nil && *doc.Decision.Locked
}

func suppressDecisionDirMissing(diagnostics diagnostic.Diagnostics) diagnostic.Diagnostics {
	var filtered diagnostic.Diagnostics
	for _, item := range diagnostics {
		if item.Code == "decision.dir_missing" {
			continue
		}
		filtered = append(filtered, item)
	}
	if filtered == nil {
		return diagnostic.Diagnostics{}
	}
	return filtered
}

func normalizeQualitySummary(summary QualitySummary) QualitySummary {
	if summary.Diagnostics == nil {
		summary.Diagnostics = diagnostic.Diagnostics{}
	}
	summary.Diagnostics.Sort()
	summary.Errors = summary.Diagnostics.Count(diagnostic.SeverityError)
	summary.Warnings = summary.Diagnostics.Count(diagnostic.SeverityWarning)
	return summary
}

func acceptedLockedDecisionCount(decisions []fileSummary) int {
	count := 0
	for _, item := range decisions {
		if item.Locked && item.Status == decisionStatusAccepted {
			count++
		}
	}
	return count
}

func supersedingDecisionCount(dir string) int {
	files, diagnostics := collectDecisionFiles(dir, []string{defaultDecisionDir})
	if diagnostics.Failed() {
		return 0
	}
	count := 0
	for _, file := range files {
		doc, _, ok := loadDecisionFile(file.fullPath, file.relPath)
		if ok && strings.TrimSpace(doc.Supersedes) != "" {
			count++
		}
	}
	return count
}

func decisionDriftCount(diagnostics diagnostic.Diagnostics) int {
	count := 0
	for _, item := range diagnostics {
		if isDecisionDriftDiagnostic(item.Code) {
			count++
		}
	}
	return count
}

func isDecisionDriftDiagnostic(code string) bool {
	switch code {
	case "decision.hash_required",
		"decision.hash_invalid",
		"decision.hash_mismatch",
		"decision.supersedes_hash_required",
		"decision.supersedes_hash_invalid",
		"decision.supersedes_hash_mismatch":
		return true
	default:
		return false
	}
}

func loadManifest(dir string) (manifest.Manifest, diagnostic.Diagnostics, bool) {
	m, err := manifest.Load(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return manifest.Manifest{}, diagnostic.Diagnostics{errorDiagnostic("decision.manifest_read_failed", "nucleus.yaml", "nucleus.yaml is required for decision validation")}, false
		}
		return manifest.Manifest{}, diagnostic.Diagnostics{errorDiagnostic("decision.manifest_read_failed", "nucleus.yaml", "nucleus.yaml could not be loaded")}, false
	}
	diagnostics := manifest.ValidateDiagnostics(m)
	return m, diagnostics, true
}

func loadDescription(dir string) (inspect.Description, diagnostic.Diagnostics, bool) {
	description, err := inspect.Describe(dir)
	if err != nil {
		return inspect.Description{}, diagnostic.Diagnostics{errorDiagnostic("decision.describe_failed", ".", safeError(err))}, false
	}
	return description, nil, true
}

func decisionsByID(decisions []loadedDecision) map[string]loadedDecision {
	byID := map[string]loadedDecision{}
	for _, item := range decisions {
		if strings.TrimSpace(item.doc.ID) != "" {
			byID[item.doc.ID] = item
		}
	}
	return byID
}

func validateDocument(item loadedDecision, m manifest.Manifest, manifestLoaded bool, description inspect.Description, descriptionLoaded bool, byID map[string]loadedDecision) diagnostic.Diagnostics {
	doc := item.doc
	path := item.path
	var diagnostics diagnostic.Diagnostics
	if strings.TrimSpace(doc.SchemaVersion) != decisionSchemaVersion {
		diagnostics = append(diagnostics, errorDiagnostic("decision.schema_version_invalid", path, "schema_version must be decision.v1"))
	}
	if strings.TrimSpace(doc.ID) == "" {
		diagnostics = append(diagnostics, errorDiagnostic("decision.id_required", path, "id is required"))
	}
	if strings.TrimSpace(doc.Capability) == "" {
		diagnostics = append(diagnostics, errorDiagnostic("decision.capability_required", path, "capability is required"))
	} else if manifestLoaded && !manifest.HasCapability(m.Capabilities, doc.Capability) {
		diagnostics = append(diagnostics, errorDiagnostic("decision.capability_missing", path, "capability is not declared in nucleus.yaml"))
	}
	diagnostics = append(diagnostics, validateChoice(path, doc)...)
	diagnostics = append(diagnostics, validateReason(path, doc.Reason)...)
	diagnostics = append(diagnostics, validateImpact(path, doc.Impact, description, descriptionLoaded)...)
	diagnostics = append(diagnostics, validateVerification(path, doc.Verification)...)
	diagnostics = append(diagnostics, validateHash(path, doc, byID)...)
	return diagnostics
}

func validateChoice(path string, doc document) diagnostic.Diagnostics {
	var diagnostics diagnostic.Diagnostics
	if strings.TrimSpace(doc.Decision.Provider) == "" && strings.TrimSpace(doc.Decision.Library) == "" && strings.TrimSpace(doc.Decision.Driver) == "" {
		diagnostics = append(diagnostics, errorDiagnostic("decision.evidence_missing", path, "decision requires provider, library, or driver evidence"))
	}
	switch strings.TrimSpace(doc.Decision.Status) {
	case decisionStatusProposed, decisionStatusAccepted, decisionStatusSupersede:
	default:
		diagnostics = append(diagnostics, errorDiagnostic("decision.status_invalid", path, "decision.status must be proposed, accepted, or superseded"))
	}
	if doc.Decision.Locked == nil {
		diagnostics = append(diagnostics, errorDiagnostic("decision.locked_required", path, "decision.locked is required"))
	}
	if doc.Decision.Locked != nil && *doc.Decision.Locked && strings.TrimSpace(doc.DecisionHash) == "" {
		diagnostics = append(diagnostics, errorDiagnostic("decision.hash_required", path, "locked decision requires decision_hash"))
	}
	return diagnostics
}

func validateReason(path string, reason []string) diagnostic.Diagnostics {
	if len(reason) == 0 {
		return diagnostic.Diagnostics{errorDiagnostic("decision.reason_required", path, "reason requires at least one item")}
	}
	var diagnostics diagnostic.Diagnostics
	for _, item := range reason {
		if strings.TrimSpace(item) == "" {
			diagnostics = append(diagnostics, errorDiagnostic("decision.reason_empty", path, "reason entries must not be empty"))
		}
	}
	return diagnostics
}

func validateImpact(path string, impact impact, description inspect.Description, loaded bool) diagnostic.Diagnostics {
	var diagnostics diagnostic.Diagnostics
	for _, file := range impact.Files {
		if invalidRelativePath(file) {
			diagnostics = append(diagnostics, errorDiagnostic("decision.impact_file_invalid", path, "impact.files entries must be relative paths inside the service directory"))
			continue
		}
		if loaded {
			surface := classifySurface(file, description.EditSurfaces)
			if surface != "allowed" {
				diagnostics = append(diagnostics, errorDiagnostic("decision.impact_file_outside_edit_surface", path, "impact file "+file+" is "+surface))
			}
		}
	}
	return diagnostics
}

func validateVerification(path string, verification verification) diagnostic.Diagnostics {
	if len(verification.Commands) == 0 {
		return diagnostic.Diagnostics{errorDiagnostic("decision.verification_required", path, "verification.commands requires at least one command")}
	}
	var diagnostics diagnostic.Diagnostics
	for _, command := range verification.Commands {
		if strings.TrimSpace(command) == "" {
			diagnostics = append(diagnostics, errorDiagnostic("decision.verification_command_empty", path, "verification.commands entries must not be empty"))
		}
	}
	return diagnostics
}

func validateHash(path string, doc document, byID map[string]loadedDecision) diagnostic.Diagnostics {
	var diagnostics diagnostic.Diagnostics
	if doc.DecisionHash != "" {
		if !validHash(doc.DecisionHash) {
			diagnostics = append(diagnostics, errorDiagnostic("decision.hash_invalid", path, "decision_hash must use algorithm:value format"))
		} else if expected := canonicalDecisionHash(doc); doc.DecisionHash != expected {
			diagnostics = append(diagnostics, errorDiagnostic("decision.hash_mismatch", path, "decision_hash does not match canonical decision payload"))
		}
	}
	if strings.TrimSpace(doc.Supersedes) == "" {
		return diagnostics
	}
	if strings.TrimSpace(doc.SupersedesHash) == "" {
		diagnostics = append(diagnostics, errorDiagnostic("decision.supersedes_hash_required", path, "supersedes requires supersedes_hash"))
		return diagnostics
	}
	if !validHash(doc.SupersedesHash) {
		diagnostics = append(diagnostics, errorDiagnostic("decision.supersedes_hash_invalid", path, "supersedes_hash must use algorithm:value format"))
		return diagnostics
	}
	previous, ok := byID[doc.Supersedes]
	if !ok {
		diagnostics = append(diagnostics, warningDiagnostic("decision.supersedes_not_loaded", path, "superseded decision was not part of this validation input"))
		return diagnostics
	}
	expected := previous.doc.DecisionHash
	if expected == "" {
		expected = canonicalDecisionHash(previous.doc)
	}
	if doc.SupersedesHash != expected {
		diagnostics = append(diagnostics, errorDiagnostic("decision.supersedes_hash_mismatch", path, "supersedes_hash does not match superseded decision"))
	}
	return diagnostics
}

func canonicalDecisionHash(doc document) string {
	payload := map[string]any{
		"capability": strings.TrimSpace(doc.Capability),
		"decision": map[string]any{
			"provider":    strings.TrimSpace(doc.Decision.Provider),
			"library":     strings.TrimSpace(doc.Decision.Library),
			"driver":      strings.TrimSpace(doc.Decision.Driver),
			"status":      strings.TrimSpace(doc.Decision.Status),
			"locked":      boolValue(doc.Decision.Locked),
			"accepted_by": strings.TrimSpace(doc.Decision.AcceptedBy),
		},
		"reason":       cleanStrings(doc.Reason),
		"impact":       map[string]any{"symbols": cleanStrings(doc.Impact.Symbols), "files": cleanStrings(doc.Impact.Files)},
		"verification": map[string]any{"commands": cleanStrings(doc.Verification.Commands)},
		"alternatives": canonicalAlternatives(doc.Alternatives),
	}
	data, _ := json.Marshal(payload)
	sum := sha256.Sum256(data)
	return decisionHashAlgorithm + ":" + base64.RawURLEncoding.EncodeToString(sum[:])
}

func canonicalAlternatives(alternatives []alternative) []map[string]string {
	result := make([]map[string]string, 0, len(alternatives))
	for _, item := range alternatives {
		result = append(result, map[string]string{
			"provider": strings.TrimSpace(item.Provider),
			"library":  strings.TrimSpace(item.Library),
			"driver":   strings.TrimSpace(item.Driver),
			"reason":   strings.TrimSpace(item.Reason),
		})
	}
	return result
}

func boolValue(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func cleanStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, strings.TrimSpace(value))
	}
	return result
}

func validHash(value string) bool {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	for _, r := range parts[0] {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func invalidRelativePath(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || filepath.IsAbs(value) || strings.HasPrefix(value, "/") {
		return true
	}
	clean := filepath.ToSlash(filepath.Clean(strings.ReplaceAll(value, "\\", "/")))
	return clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../")
}

func classifySurface(path string, surfaces inspect.EditSurfaces) string {
	switch {
	case matchesAnySurface(path, surfaces.Forbidden):
		return "forbidden"
	case matchesAnySurface(path, surfaces.Readonly):
		return "readonly"
	case matchesAnySurface(path, surfaces.Allowed):
		return "allowed"
	default:
		return "manual"
	}
}

func matchesAnySurface(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if surfaceMatch(path, pattern) {
			return true
		}
	}
	return false
}

func surfaceMatch(path string, pattern string) bool {
	pattern = strings.TrimPrefix(strings.ReplaceAll(pattern, "\\", "/"), "./")
	path = strings.TrimPrefix(strings.ReplaceAll(path, "\\", "/"), "./")
	if pattern == path || pattern == "**" {
		return true
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*") + "/"
		rest := strings.TrimPrefix(path, prefix)
		return strings.HasPrefix(path, prefix) && rest != "" && !strings.Contains(rest, "/")
	}
	return false
}

func safeError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func errorDiagnostic(code string, path string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{Severity: diagnostic.SeverityError, Code: code, Path: path, Message: message}
}

func warningDiagnostic(code string, path string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{Severity: diagnostic.SeverityWarning, Code: code, Path: path, Message: message}
}
