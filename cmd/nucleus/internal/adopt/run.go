package adopt

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
)

func run(config Config, opts *options) (result, error) {
	dir := stringValue(config.Dir, defaultDir)
	facts := scan(dir)
	normalized := normalizeOptions(opts, facts, dir)
	diagnostics := append(diagnostic.Diagnostics{}, facts.diagnostics...)
	if err := validateOptions(normalized); err != nil {
		diagnostics = append(diagnostics, errorDiagnostic("adopt.invalid_options", "", err.Error()))
		return buildResult(facts, nil, diagnostics), err
	}

	created, writeDiagnostics, err := writeProtocolIndex(dir, normalized)
	diagnostics = append(diagnostics, writeDiagnostics...)
	diagnostics.Sort()
	output := buildResult(facts, created, diagnostics)
	if err != nil {
		return output, err
	}
	return output, nil
}

type normalizedOptions struct {
	service string
	version string
	intent  string
	agent   string
	force   bool
}

type projectFacts struct {
	module                  string
	service                 string
	packages                []string
	contracts               []pathFact
	testCommands            []string
	generatedFileCandidates []pathFact
	symbols                 map[string]int
	diagnostics             diagnostic.Diagnostics
}

func normalizeOptions(opts *options, facts projectFacts, dir string) normalizedOptions {
	if opts == nil {
		opts = &options{}
	}
	service := strings.TrimSpace(opts.service)
	if service == "" {
		service = facts.service
	}
	if service == "" {
		service = inferredServiceName(dir, facts.module)
	}
	version := strings.TrimSpace(opts.version)
	if version == "" {
		version = defaultVersion
	}
	intent := strings.TrimSpace(opts.intent)
	if intent == "" {
		intent = defaultIntent
	}
	return normalizedOptions{
		service: service,
		version: version,
		intent:  intent,
		agent:   strings.TrimSpace(opts.agent),
		force:   opts.force,
	}
}

func validateOptions(opts normalizedOptions) error {
	switch opts.agent {
	case "", agentCodex:
	default:
		return fmt.Errorf("unknown agent %q", opts.agent)
	}
	if strings.TrimSpace(opts.service) == "" {
		return errors.New("service name is required")
	}
	if strings.ContainsAny(opts.service, "\n\r\t") {
		return errors.New("service name must be a single line")
	}
	return nil
}

func writeProtocolIndex(dir string, opts normalizedOptions) ([]pathFact, diagnostic.Diagnostics, error) {
	files := map[string]string{
		manifestFileName:  manifestYAML(opts),
		decisionsKeepFile: "",
		nucleusReadmeFile: nucleusReadme(),
	}
	if opts.agent == agentCodex {
		files[codexInstructionFile] = codexInstruction()
	}

	var diagnostics diagnostic.Diagnostics
	created := make([]pathFact, 0, len(files))
	for _, name := range sortedKeys(files) {
		pathValue := filepath.Join(dir, filepath.FromSlash(name))
		if !opts.force {
			if _, err := os.Stat(pathValue); err == nil {
				diagnostics = append(diagnostics, warningDiagnostic("adopt.file_exists", name, "existing protocol file was left unchanged"))
				continue
			} else if err != nil && !os.IsNotExist(err) {
				return created, append(diagnostics, errorDiagnostic("adopt.stat_failed", name, err.Error())), err
			}
		}
		if err := os.MkdirAll(filepath.Dir(pathValue), 0o755); err != nil {
			return created, append(diagnostics, errorDiagnostic("adopt.mkdir_failed", name, err.Error())), err
		}
		if err := os.WriteFile(pathValue, []byte(files[name]), 0o644); err != nil {
			return created, append(diagnostics, errorDiagnostic("adopt.write_failed", name, err.Error())), err
		}
		created = append(created, pathFact{Path: name, Kind: "protocol_index", Reason: "adopt"})
	}
	return created, diagnostics, nil
}

func buildResult(facts projectFacts, created []pathFact, diagnostics diagnostic.Diagnostics) result {
	return normalizeResult(result{
		ResultKind:              resultKindAdopt,
		SchemaVersion:           schemaVersionAdopt,
		SchemaRef:               schemaRefAdopt,
		OK:                      !diagnostics.Failed(),
		DetectedModule:          facts.module,
		PackageSummary:          facts.packages,
		DetectedContracts:       facts.contracts,
		DetectedTestCommands:    facts.testCommands,
		CreatedFiles:            created,
		GeneratedFileCandidates: facts.generatedFileCandidates,
		SymbolIndexSummary:      facts.symbols,
		Diagnostics:             diagnostics,
	})
}

func scan(dir string) projectFacts {
	module, moduleDiagnostics := detectModule(dir)
	facts := projectFacts{
		module:                  module,
		service:                 inferredServiceName(dir, module),
		contracts:               detectContracts(dir),
		testCommands:            detectTestCommands(dir, module),
		generatedFileCandidates: []pathFact{},
		symbols:                 map[string]int{},
		diagnostics:             moduleDiagnostics,
	}
	packageNames := map[string]string{}
	symbols := map[string]int{
		"packages":  0,
		"files":     0,
		"functions": 0,
		"methods":   0,
		"types":     0,
		"symbols":   0,
	}
	if info, err := os.Stat(dir); err != nil {
		if !os.IsNotExist(err) {
			facts.diagnostics = append(facts.diagnostics, warningDiagnostic("adopt.scan_skipped", ".", err.Error()))
		}
		facts.symbols = symbols
		facts.diagnostics.Sort()
		return facts
	} else if !info.IsDir() {
		facts.diagnostics = append(facts.diagnostics, warningDiagnostic("adopt.scan_skipped", ".", "target path is not a directory"))
		facts.symbols = symbols
		facts.diagnostics.Sort()
		return facts
	}
	_ = filepath.WalkDir(dir, func(filePath string, entry os.DirEntry, err error) error {
		if err != nil {
			facts.diagnostics = append(facts.diagnostics, warningDiagnostic("adopt.scan_skipped", safeRel(dir, filePath), err.Error()))
			return nil
		}
		if entry.IsDir() {
			if shouldSkipDir(entry.Name()) && filePath != dir {
				return filepath.SkipDir
			}
			return nil
		}
		rel := safeRel(dir, filePath)
		if generatedCandidate(rel) {
			facts.generatedFileCandidates = append(facts.generatedFileCandidates, pathFact{Path: rel, Kind: "generated_candidate", Reason: "name or path suggests generated code"})
		}
		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		parsed, parseErr := parser.ParseFile(token.NewFileSet(), filePath, nil, 0)
		if parseErr != nil {
			facts.diagnostics = append(facts.diagnostics, warningDiagnostic("adopt.go_parse_skipped", rel, parseErr.Error()))
			return nil
		}
		symbols["files"]++
		packagePath := path.Dir(rel)
		if packagePath == "." {
			packagePath = "."
		}
		packageNames[packagePath] = parsed.Name.Name
		ast.Inspect(parsed, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.FuncDecl:
				if typed.Recv == nil {
					symbols["functions"]++
				} else {
					symbols["methods"]++
				}
				symbols["symbols"]++
			case *ast.TypeSpec:
				symbols["types"]++
				symbols["symbols"]++
			}
			return true
		})
		return nil
	})
	for packagePath, packageName := range packageNames {
		facts.packages = append(facts.packages, packagePath+" "+packageName)
	}
	sort.Strings(facts.packages)
	sort.Slice(facts.generatedFileCandidates, func(i, j int) bool {
		return facts.generatedFileCandidates[i].Path < facts.generatedFileCandidates[j].Path
	})
	symbols["packages"] = len(facts.packages)
	facts.symbols = symbols
	facts.diagnostics.Sort()
	return facts
}

func detectModule(dir string) (string, diagnostic.Diagnostics) {
	file, err := os.Open(filepath.Join(dir, "go.mod"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", diagnostic.Diagnostics{warningDiagnostic("adopt.go_mod_missing", "go.mod", "go.mod was not found; service name was inferred from directory")}
		}
		return "", diagnostic.Diagnostics{warningDiagnostic("adopt.go_mod_read_failed", "go.mod", err.Error())}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return fields[1], nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", diagnostic.Diagnostics{warningDiagnostic("adopt.go_mod_read_failed", "go.mod", err.Error())}
	}
	return "", diagnostic.Diagnostics{warningDiagnostic("adopt.go_mod_module_missing", "go.mod", "go.mod does not declare a module path")}
}

func detectContracts(dir string) []pathFact {
	candidates := []pathFact{
		{Path: "api/openapi.yaml", Kind: "openapi"},
		{Path: "api/openapi.yml", Kind: "openapi"},
		{Path: "api/errors.yaml", Kind: "errors"},
	}
	var facts []pathFact
	for _, candidate := range candidates {
		if regularFile(filepath.Join(dir, filepath.FromSlash(candidate.Path))) {
			candidate.Reason = "well-known contract path"
			facts = append(facts, candidate)
		}
	}
	protoRoot := filepath.Join(dir, "api", "proto")
	_ = filepath.WalkDir(protoRoot, func(filePath string, entry os.DirEntry, err error) error {
		if err != nil || entry == nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".proto") {
			return nil
		}
		facts = append(facts, pathFact{Path: safeRel(dir, filePath), Kind: "proto", Reason: "proto contract candidate"})
		return nil
	})
	sort.Slice(facts, func(i, j int) bool {
		return facts[i].Path < facts[j].Path
	})
	return facts
}

func detectTestCommands(dir string, module string) []string {
	var commands []string
	if module != "" || regularFile(filepath.Join(dir, "go.mod")) {
		commands = append(commands, "go test ./...")
	}
	if regularFile(filepath.Join(dir, "api", "openapi.yaml")) || regularFile(filepath.Join(dir, "api", "openapi.yml")) || regularFile(filepath.Join(dir, "api", "errors.yaml")) {
		commands = append(commands, "nucleus validate --dir .")
	}
	return commands
}

func manifestYAML(opts normalizedOptions) string {
	return strings.Join([]string{
		"schema_version: " + quoteYAML("2.0"),
		"service:",
		"  name: " + quoteYAML(opts.service),
		"  version: " + quoteYAML(opts.version),
		"ai:",
		"  intent: " + quoteYAML(opts.intent),
		"  allowed_changes:",
		"    - " + quoteYAML("nucleus.yaml"),
		"    - " + quoteYAML(".nucleus/**"),
		"    - " + quoteYAML("api/**"),
		"    - " + quoteYAML("docs/**"),
		"  readonly: []",
		"  forbidden:",
		"    - " + quoteYAML("configs/*.local.yaml"),
		"    - " + quoteYAML("configs/*.secret.yaml"),
		"    - " + quoteYAML(".env"),
		"    - " + quoteYAML(".env.*"),
		"capabilities: []",
		"dependencies: []",
		"verify:",
		"  commands:",
		"    - " + quoteYAML("go test ./..."),
		"",
	}, "\n")
}

func nucleusReadme() string {
	return "# Nucleus Protocol Index\n" +
		"\n" +
		"This directory stores local, agent-readable protocol metadata for the project.\n" +
		"\n" +
		"- Provider, library, and driver choices belong in `.nucleus/decisions`.\n" +
		"- Generated or inferred facts should be rebuilt with `nucleus describe`.\n" +
		"- This directory must not contain business implementation code.\n"
}

func codexInstruction() string {
	return "# Codex Agent Notes\n" +
		"\n" +
		"Use Nucleus as a protocol layer, not as a project scaffold.\n" +
		"\n" +
		"- Inspect with `nucleus describe --dir . --json --flow` before behavior changes.\n" +
		"- Keep provider/library/driver choices in structured decision evidence.\n" +
		"- Do not modify go.mod or go.sum unless the user explicitly approves the dependency decision.\n" +
		"- Prefer existing project structure and interfaces over new framework-shaped directories.\n"
}

func quoteYAML(value string) string {
	data, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(data)
}

func inferredServiceName(dir string, module string) string {
	if module != "" {
		if name := path.Base(module); name != "." && name != "/" {
			return name
		}
	}
	name := filepath.Base(filepath.Clean(dir))
	if name == "." || name == string(filepath.Separator) {
		return "unknown"
	}
	return name
}

func generatedCandidate(rel string) bool {
	rel = strings.ReplaceAll(rel, "\\", "/")
	base := path.Base(rel)
	return strings.Contains(rel, "/gen/") || strings.Contains(rel, "/generated/") || strings.HasSuffix(base, ".gen.go") || strings.HasSuffix(base, ".pb.go")
}

func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", ".idea", ".vscode", "vendor", "node_modules", ".nucleus":
		return true
	default:
		return false
	}
}

func safeRel(dir string, filePath string) string {
	rel, err := filepath.Rel(dir, filePath)
	if err != nil {
		return filepath.ToSlash(filePath)
	}
	return filepath.ToSlash(rel)
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func warningDiagnostic(code string, path string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity: diagnostic.SeverityWarning,
		Code:     code,
		Path:     path,
		Message:  message,
	}
}

func errorDiagnostic(code string, path string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity: diagnostic.SeverityError,
		Code:     code,
		Path:     path,
		Message:  message,
	}
}
