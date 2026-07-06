package recipe

import (
	"bytes"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
	"go.yaml.in/yaml/v3"
)

const (
	defaultRecipeDir        = ".nucleus/recipes"
	builtinRecipeDir        = "builtin"
	builtinRecipePathPrefix = "builtin://recipes/"
	recipeSourceBuiltin     = "builtin"
	recipeSourceProject     = "project"
	schemaVersionResult     = "recipe-result.v1"
	schemaRefResult         = "contract/schema/recipe-result.v1.schema.json"
)

//go:embed builtin/*.yaml
var builtinRecipes embed.FS

// Filter selects recipes by stable metadata.
type Filter struct {
	Kind     string
	Provider string
}

// Document is a read-only recipe knowledge document.
type Document struct {
	SchemaVersion string   `yaml:"schema_version" json:"schema_version"`
	ID            string   `yaml:"id" json:"id"`
	Kind          string   `yaml:"kind" json:"kind"`
	Provider      string   `yaml:"provider,omitempty" json:"provider,omitempty"`
	Language      string   `yaml:"language" json:"language"`
	Detect        Detect   `yaml:"detect,omitempty" json:"detect,omitempty"`
	Suggest       Suggest  `yaml:"suggest,omitempty" json:"suggest,omitempty"`
	Risks         []string `yaml:"risks,omitempty" json:"risks,omitempty"`
}

// Detect describes facts that can make a recipe relevant.
type Detect struct {
	Imports []string `yaml:"imports,omitempty" json:"imports,omitempty"`
	Files   []string `yaml:"files,omitempty" json:"files,omitempty"`
}

// Suggest carries non-executable hints for agents.
type Suggest struct {
	Interfaces   []string `yaml:"interfaces,omitempty" json:"interfaces,omitempty"`
	Verification []string `yaml:"verification,omitempty" json:"verification,omitempty"`
}

// Summary is the short list view returned by MCP and planning.
type Summary struct {
	Path     string `json:"path"`
	Source   string `json:"source"`
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Provider string `json:"provider,omitempty"`
	Language string `json:"language"`
}

// ListResult is a structured recipe list payload.
type ListResult struct {
	ResultKind    string                 `json:"result_kind"`
	SchemaVersion string                 `json:"schema_version"`
	SchemaRef     string                 `json:"schema_ref"`
	OK            bool                   `json:"ok"`
	Recipes       []Summary              `json:"recipes"`
	Diagnostics   diagnostic.Diagnostics `json:"diagnostics"`
}

// GetResult is a structured single recipe payload.
type GetResult struct {
	ResultKind    string                 `json:"result_kind"`
	SchemaVersion string                 `json:"schema_version"`
	SchemaRef     string                 `json:"schema_ref"`
	OK            bool                   `json:"ok"`
	Path          string                 `json:"path,omitempty"`
	Source        string                 `json:"source,omitempty"`
	Recipe        Document               `json:"recipe,omitempty"`
	Diagnostics   diagnostic.Diagnostics `json:"diagnostics"`
}

// CandidateQuery controls plan-time recipe matching.
type CandidateQuery struct {
	Task  string
	Kinds []string
	Limit int
}

// CandidateResult contains read-only recipe candidates for plan context.
type CandidateResult struct {
	ResultKind    string                 `json:"result_kind"`
	SchemaVersion string                 `json:"schema_version"`
	SchemaRef     string                 `json:"schema_ref"`
	OK            bool                   `json:"ok"`
	Candidates    []Candidate            `json:"candidates"`
	Diagnostics   diagnostic.Diagnostics `json:"diagnostics"`
}

// Candidate is a recipe hint. It is never an accepted decision.
type Candidate struct {
	Path                  string   `json:"path"`
	Source                string   `json:"source"`
	ID                    string   `json:"id"`
	Kind                  string   `json:"kind"`
	Provider              string   `json:"provider,omitempty"`
	Language              string   `json:"language"`
	Match                 []string `json:"match"`
	SuggestedInterfaces   []string `json:"suggested_interfaces,omitempty"`
	SuggestedVerification []string `json:"suggested_verification,omitempty"`
	Risks                 []string `json:"risks,omitempty"`
	Selection             string   `json:"selection"`
	DecisionRequired      bool     `json:"decision_required"`
}

// List returns strict, read-only recipe summaries.
func List(dir string, filter Filter) (ListResult, error) {
	recipesWithSource, diagnostics := loadRecipeEntries(dir)
	var recipes []Summary
	for _, item := range recipesWithSource {
		if !recipeMatches(item.doc, filter) {
			continue
		}
		recipes = append(recipes, Summary{Path: item.file.relPath, Source: item.file.source, ID: item.doc.ID, Kind: item.doc.Kind, Provider: item.doc.Provider, Language: item.doc.Language})
	}
	sort.Slice(recipes, func(i, j int) bool { return recipes[i].ID < recipes[j].ID })
	diagnostics.Sort()
	if recipes == nil {
		recipes = []Summary{}
	}
	if diagnostics == nil {
		diagnostics = diagnostic.Diagnostics{}
	}
	return ListResult{
		ResultKind:    "nucleus.recipe_list_result",
		SchemaVersion: schemaVersionResult,
		SchemaRef:     schemaRefResult,
		OK:            !diagnostics.Failed(),
		Recipes:       recipes,
		Diagnostics:   diagnostics,
	}, nil
}

// Get returns one strict, read-only recipe document.
func Get(dir string, id string) (GetResult, error) {
	files, diagnostics := collectRecipeFiles(dir)
	for _, file := range files {
		doc, itemDiagnostics, ok := loadRecipeFile(file)
		diagnostics = append(diagnostics, itemDiagnostics...)
		if doc.ID == id {
			diagnostics.Sort()
			if diagnostics == nil {
				diagnostics = diagnostic.Diagnostics{}
			}
			return GetResult{
				ResultKind:    "nucleus.recipe_result",
				SchemaVersion: schemaVersionResult,
				SchemaRef:     schemaRefResult,
				OK:            ok && !diagnostics.Failed(),
				Path:          file.relPath,
				Source:        file.source,
				Recipe:        doc,
				Diagnostics:   diagnostics,
			}, nil
		}
	}
	diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.SeverityError, Code: "recipe.not_found", Path: defaultRecipeDir, Message: "recipe id was not found"})
	diagnostics.Sort()
	return GetResult{
		ResultKind:    "nucleus.recipe_result",
		SchemaVersion: schemaVersionResult,
		SchemaRef:     schemaRefResult,
		OK:            false,
		Diagnostics:   diagnostics,
	}, nil
}

// Candidates returns read-only plan hints. Missing recipe directories are not a plan warning.
func Candidates(dir string, query CandidateQuery) CandidateResult {
	recipesWithSource, diagnostics := loadRecipeEntries(dir)
	diagnostics = suppressMissingDir(diagnostics)
	var candidates []Candidate
	for _, item := range recipesWithSource {
		doc := item.doc
		matches := candidateMatches(dir, doc, query)
		if len(matches) == 0 {
			continue
		}
		candidates = append(candidates, Candidate{
			Path:                  item.file.relPath,
			Source:                item.file.source,
			ID:                    doc.ID,
			Kind:                  doc.Kind,
			Provider:              doc.Provider,
			Language:              doc.Language,
			Match:                 matches,
			SuggestedInterfaces:   nonNilStrings(doc.Suggest.Interfaces),
			SuggestedVerification: nonNilStrings(doc.Suggest.Verification),
			Risks:                 nonNilStrings(doc.Risks),
			Selection:             "candidate_only",
			DecisionRequired:      true,
		})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	if query.Limit > 0 && len(candidates) > query.Limit {
		candidates = candidates[:query.Limit]
	}
	diagnostics.Sort()
	if candidates == nil {
		candidates = []Candidate{}
	}
	if diagnostics == nil {
		diagnostics = diagnostic.Diagnostics{}
	}
	return CandidateResult{
		ResultKind:    "nucleus.recipe_candidate_result",
		SchemaVersion: schemaVersionResult,
		SchemaRef:     schemaRefResult,
		OK:            !diagnostics.Failed(),
		Candidates:    candidates,
		Diagnostics:   diagnostics,
	}
}

type recipeFile struct {
	relPath   string
	fullPath  string
	embedPath string
	source    string
}

type loadedRecipe struct {
	file recipeFile
	doc  Document
}

func loadRecipeEntries(dir string) ([]loadedRecipe, diagnostic.Diagnostics) {
	files, diagnostics := collectRecipeFiles(dir)
	var loaded []loadedRecipe
	projectIDs := map[string]bool{}
	for _, file := range files {
		doc, itemDiagnostics, ok := loadRecipeFile(file)
		diagnostics = append(diagnostics, itemDiagnostics...)
		if file.source == recipeSourceProject && strings.TrimSpace(doc.ID) != "" {
			projectIDs[doc.ID] = true
		}
		if !ok {
			continue
		}
		loaded = append(loaded, loadedRecipe{file: file, doc: doc})
	}
	var selected []loadedRecipe
	seenIDs := map[string]bool{}
	for _, item := range loaded {
		if item.file.source == recipeSourceBuiltin && projectIDs[item.doc.ID] {
			continue
		}
		if seenIDs[item.doc.ID] {
			continue
		}
		seenIDs[item.doc.ID] = true
		selected = append(selected, item)
	}
	return selected, diagnostics
}

func collectRecipeFiles(dir string) ([]recipeFile, diagnostic.Diagnostics) {
	projectFiles, diagnostics := collectProjectRecipeFiles(dir)
	builtinFiles, builtinDiagnostics := collectBuiltinRecipeFiles()
	diagnostics = append(diagnostics, builtinDiagnostics...)
	files := append(projectFiles, builtinFiles...)
	sort.Slice(files, func(i, j int) bool {
		if files[i].source != files[j].source {
			return files[i].source == recipeSourceProject
		}
		return files[i].relPath < files[j].relPath
	})
	return files, diagnostics
}

func collectProjectRecipeFiles(dir string) ([]recipeFile, diagnostic.Diagnostics) {
	root := filepath.Join(dir, filepath.FromSlash(defaultRecipeDir))
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, diagnostic.Diagnostics{diagnostic.Diagnostic{Severity: diagnostic.SeverityWarning, Code: "recipe.dir_missing", Path: defaultRecipeDir, Message: "recipe directory does not exist"}}
		}
		return nil, diagnostic.Diagnostics{diagnostic.Diagnostic{Severity: diagnostic.SeverityError, Code: "recipe.dir_read_failed", Path: defaultRecipeDir, Message: err.Error()}}
	}
	if !info.IsDir() {
		return nil, diagnostic.Diagnostics{diagnostic.Diagnostic{Severity: diagnostic.SeverityError, Code: "recipe.dir_invalid", Path: defaultRecipeDir, Message: "recipe path is not a directory"}}
	}
	var files []recipeFile
	var diagnostics diagnostic.Diagnostics
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.SeverityWarning, Code: "recipe.path_read_failed", Path: defaultRecipeDir, Message: err.Error()})
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			return nil
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.SeverityWarning, Code: "recipe.path_read_failed", Path: defaultRecipeDir, Message: relErr.Error()})
			return nil
		}
		files = append(files, recipeFile{relPath: filepath.ToSlash(rel), fullPath: path, source: recipeSourceProject})
		return nil
	})
	if err != nil {
		diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.SeverityError, Code: "recipe.dir_read_failed", Path: defaultRecipeDir, Message: err.Error()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].relPath < files[j].relPath })
	return files, diagnostics
}

func collectBuiltinRecipeFiles() ([]recipeFile, diagnostic.Diagnostics) {
	var files []recipeFile
	err := fs.WalkDir(builtinRecipes, builtinRecipeDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			return nil
		}
		name := strings.TrimPrefix(path, builtinRecipeDir+"/")
		files = append(files, recipeFile{
			relPath:   builtinRecipePathPrefix + filepath.ToSlash(name),
			embedPath: path,
			source:    recipeSourceBuiltin,
		})
		return nil
	})
	if err != nil {
		return nil, diagnostic.Diagnostics{diagnostic.Diagnostic{Severity: diagnostic.SeverityError, Code: "recipe.builtin_read_failed", Path: builtinRecipeDir, Message: err.Error()}}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].relPath < files[j].relPath })
	return files, nil
}

func loadRecipeFile(file recipeFile) (Document, diagnostic.Diagnostics, bool) {
	data, err := readRecipeFile(file)
	if err != nil {
		return Document{}, diagnostic.Diagnostics{diagnostic.Diagnostic{Severity: diagnostic.SeverityError, Code: "recipe.read_failed", Path: file.relPath, Message: err.Error()}}, false
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var doc Document
	if err := decoder.Decode(&doc); err != nil {
		return Document{}, diagnostic.Diagnostics{diagnostic.Diagnostic{Severity: diagnostic.SeverityError, Code: "recipe.parse_failed", Path: file.relPath, Message: err.Error()}}, false
	}
	diagnostics := validateRecipe(file.relPath, doc)
	return doc, diagnostics, !diagnostics.Failed()
}

func readRecipeFile(file recipeFile) ([]byte, error) {
	if file.source == recipeSourceBuiltin {
		return builtinRecipes.ReadFile(file.embedPath)
	}
	return os.ReadFile(file.fullPath)
}

func validateRecipe(path string, doc Document) diagnostic.Diagnostics {
	var diagnostics diagnostic.Diagnostics
	if doc.SchemaVersion != "recipe.v1" {
		diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.SeverityError, Code: "recipe.schema_version_invalid", Path: path, Message: "schema_version must be recipe.v1"})
	}
	if strings.TrimSpace(doc.ID) == "" {
		diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.SeverityError, Code: "recipe.id_required", Path: path, Message: "id is required"})
	}
	if strings.TrimSpace(doc.Kind) == "" {
		diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.SeverityError, Code: "recipe.kind_required", Path: path, Message: "kind is required"})
	}
	if strings.TrimSpace(doc.Language) == "" {
		diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.SeverityError, Code: "recipe.language_required", Path: path, Message: "language is required"})
	}
	for _, file := range doc.Detect.Files {
		if invalidRecipePath(file) {
			diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.SeverityError, Code: "recipe.detect_file_invalid", Path: path, Message: "detect.files entries must be relative paths inside the service directory"})
		}
	}
	return diagnostics
}

func invalidRecipePath(value string) bool {
	value = strings.TrimSpace(filepath.ToSlash(value))
	return value == "" || strings.HasPrefix(value, "/") || value == ".." || strings.HasPrefix(value, "../") || strings.Contains(value, "/../")
}

func recipeMatches(doc Document, filter Filter) bool {
	if filter.Kind != "" && doc.Kind != filter.Kind {
		return false
	}
	if filter.Provider != "" && doc.Provider != filter.Provider {
		return false
	}
	return true
}

func candidateMatches(dir string, doc Document, query CandidateQuery) []string {
	var matches []string
	for _, kind := range query.Kinds {
		if doc.Kind == kind {
			matches = append(matches, "kind:"+kind)
		}
	}
	task := " " + strings.ToLower(query.Task) + " "
	for _, token := range []string{doc.ID, doc.Kind, doc.Provider} {
		token = strings.TrimSpace(strings.ToLower(token))
		if token != "" && strings.Contains(task, token) {
			matches = append(matches, "task:"+token)
		}
	}
	for _, file := range doc.Detect.Files {
		if fileExists(dir, file) {
			matches = append(matches, "file:"+filepath.ToSlash(file))
		}
	}
	for _, item := range doc.Detect.Imports {
		if importExists(dir, item) {
			matches = append(matches, "import:"+item)
		}
	}
	return uniqueStrings(matches)
}

func suppressMissingDir(diagnostics diagnostic.Diagnostics) diagnostic.Diagnostics {
	var filtered diagnostic.Diagnostics
	for _, item := range diagnostics {
		if item.Code == "recipe.dir_missing" {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func fileExists(dir string, rel string) bool {
	path := filepath.Join(dir, filepath.FromSlash(rel))
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func importExists(dir string, importPath string) bool {
	importPath = strings.TrimSpace(importPath)
	if importPath == "" {
		return false
	}
	found := false
	_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || found || entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr == nil && bytes.Contains(data, []byte(`"`+importPath+`"`)) {
			found = true
		}
		return nil
	})
	return found
}

func nonNilStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return values
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
