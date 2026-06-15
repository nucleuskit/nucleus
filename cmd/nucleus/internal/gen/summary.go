package gen

import (
	"sort"

	"github.com/nucleuskit/contract/diagnostic"
)

type genSummary struct {
	Files           int      `json:"files"`
	Targets         []string `json:"targets"`
	ClientLanguages []string `json:"client_languages"`
	Errors          int      `json:"errors"`
	Warnings        int      `json:"warnings"`
}

func buildSummary(files []string, targets []string, clientLanguages []string, diagnostics diagnostic.Diagnostics) genSummary {
	sort.Strings(targets)
	sort.Strings(clientLanguages)
	return genSummary{
		Files:           len(files),
		Targets:         targets,
		ClientLanguages: clientLanguages,
		Errors:          diagnostics.Count(diagnostic.SeverityError),
		Warnings:        diagnostics.Count(diagnostic.SeverityWarning),
	}
}
