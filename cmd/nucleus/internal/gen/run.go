package gen

import (
	"fmt"
	"sort"

	"github.com/nucleuskit/contract/diagnostic"
	contractgen "github.com/nucleuskit/contract/gen"
	"github.com/nucleuskit/contract/inspect"
	"github.com/nucleuskit/contract/validation"
)

func run(config Config, opts *options) (genResult, error) {
	dir := stringValue(config.Dir, defaultDir)
	diagnostics := validation.ValidateService(dir)
	if diagnostics.Failed() {
		var files []string
		result := genResult{
			ResultKind:  resultKindGen,
			OK:          false,
			Summary:     buildSummary(files, nil, nil, diagnostics),
			Files:       files,
			Diagnostics: diagnostics,
		}
		return result, ErrGenFailed
	}

	files := []string{}
	targets := []string{}
	clientLanguages := []string{}
	sourceHash := ""

	if shouldGenerateCore(opts) {
		generated, err := contractgen.GenerateWithOptions(dir, coreOptions(opts))
		if err != nil {
			return failedResult(files, targets, clientLanguages, diagnostics, err), fmt.Errorf("%w: %v", ErrGenFailed, err)
		}
		sourceHash = generated.Hash
		files = append(files, relativeFiles(dir, generated.Files)...)
		targets = appendUnique(targets, coreTargets(opts)...)
	}
	if sourceHash == "" {
		hash, err := inspect.ContractSourceHash(dir)
		if err != nil {
			return failedResult(files, targets, clientLanguages, diagnostics, err), fmt.Errorf("%w: %v", ErrGenFailed, err)
		}
		sourceHash = hash
	}

	if opts.docs {
		docs, err := contractgen.ExportDocs(dir)
		if err != nil {
			return failedResult(files, targets, clientLanguages, diagnostics, err), fmt.Errorf("%w: %v", ErrGenFailed, err)
		}
		path, err := writeBytesFile(dir, contractDocsPath(), docs)
		if err != nil {
			return failedResult(files, targets, clientLanguages, diagnostics, err), fmt.Errorf("%w: %v", ErrGenFailed, err)
		}
		files = append(files, path)
		targets = appendUnique(targets, targetContractGen)
	}
	if opts.typeScript {
		types, err := contractgen.ExportTypeScript(dir)
		if err != nil {
			return failedResult(files, targets, clientLanguages, diagnostics, err), fmt.Errorf("%w: %v", ErrGenFailed, err)
		}
		path, err := writeBytesFile(dir, typeScriptSchemaPath(), types)
		if err != nil {
			return failedResult(files, targets, clientLanguages, diagnostics, err), fmt.Errorf("%w: %v", ErrGenFailed, err)
		}
		files = append(files, path)
		targets = appendUnique(targets, targetContractGen)
	}
	if opts.docs || opts.typeScript {
		if marker, err := writeFreshnessMarker(dir, targetContractGen, sourceHash); err != nil {
			return failedResult(files, targets, clientLanguages, diagnostics, err), fmt.Errorf("%w: %v", ErrGenFailed, err)
		} else if marker != "" {
			files = append(files, marker)
		}
	}
	if opts.clients {
		clients, err := contractgen.ExportClientBundle(dir, opts.clientLanguages)
		if err != nil {
			return failedResult(files, targets, clientLanguages, diagnostics, err), fmt.Errorf("%w: %v", ErrGenFailed, err)
		}
		languages := make([]string, 0, len(clients))
		for language := range clients {
			languages = append(languages, language)
		}
		sort.Strings(languages)
		for _, language := range languages {
			path, err := writeBytesFile(dir, clientOutputPath(language), clients[language])
			if err != nil {
				return failedResult(files, targets, clientLanguages, diagnostics, err), fmt.Errorf("%w: %v", ErrGenFailed, err)
			}
			files = append(files, path)
			if marker, err := writeFreshnessMarker(dir, clientTarget(language), sourceHash); err != nil {
				return failedResult(files, targets, clientLanguages, diagnostics, err), fmt.Errorf("%w: %v", ErrGenFailed, err)
			} else if marker != "" {
				files = append(files, marker)
			}
			clientLanguages = appendUnique(clientLanguages, language)
			targets = appendUnique(targets, clientTarget(language))
		}
	}

	files = appendUnique(nil, files...)
	sort.Strings(files)
	result := genResult{
		ResultKind:  resultKindGen,
		OK:          true,
		SourceHash:  sourceHash,
		Summary:     buildSummary(files, targets, clientLanguages, diagnostics),
		Files:       files,
		Diagnostics: diagnostics,
	}
	return result, nil
}

func shouldGenerateCore(opts *options) bool {
	return opts.http || opts.grpc || opts.errors || (!opts.docs && !opts.typeScript && !opts.clients)
}

func coreOptions(opts *options) contractgen.Options {
	if !opts.http && !opts.grpc && !opts.errors {
		return contractgen.Options{}
	}
	return contractgen.Options{
		HTTP:   opts.http,
		GRPC:   opts.grpc,
		Errors: opts.errors,
	}
}

func coreTargets(opts *options) []string {
	if !opts.http && !opts.grpc && !opts.errors {
		return []string{targetContractGen, targetHTTPGen}
	}
	targets := []string{targetContractGen}
	if opts.http {
		targets = append(targets, targetHTTPGen)
	}
	return targets
}

func failedResult(files []string, targets []string, clientLanguages []string, diagnostics diagnostic.Diagnostics, err error) genResult {
	diagnostics = append(diagnostics, diagnostic.Diagnostic{
		Severity: diagnostic.SeverityError,
		Code:     "gen.failed",
		Message:  err.Error(),
	})
	diagnostics.Sort()
	files = appendUnique(nil, files...)
	sort.Strings(files)
	return genResult{
		ResultKind:  resultKindGen,
		OK:          false,
		Summary:     buildSummary(files, targets, clientLanguages, diagnostics),
		Files:       files,
		Diagnostics: diagnostics,
	}
}
