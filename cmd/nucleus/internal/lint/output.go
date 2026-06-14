package lint

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	contractlint "github.com/nucleuskit/contract/lint"
)

type lintResult struct {
	ResultKind string                 `json:"result_kind"`
	OK         bool                   `json:"ok"`
	Summary    lintSummary            `json:"summary"`
	Findings   []contractlint.Finding `json:"findings"`
	Strict     bool                   `json:"strict"`
}

func renderHuman(stdout io.Writer, stderr io.Writer, findings []contractlint.Finding, summary lintSummary) {
	for _, finding := range findings {
		if finding.Path == "" {
			_, _ = fmt.Fprintf(stderr, "error %s: %s\n", finding.Rule, finding.Message)
			continue
		}
		_, _ = fmt.Fprintf(stderr, "error %s %s: %s\n", finding.Path, finding.Rule, finding.Message)
	}
	if len(findings) > 0 {
		return
	}
	_, _ = fmt.Fprintln(stdout, "OK")
	if len(summary.Checked) > 0 {
		_, _ = fmt.Fprintf(stdout, "linted: %s\n", strings.Join(summary.Checked, ", "))
	}
	if summary.Mode != "" {
		_, _ = fmt.Fprintf(stdout, "mode: %s\n", summary.Mode)
	}
	if len(summary.ActiveRules) > 0 {
		_, _ = fmt.Fprintf(stdout, "rules: %s\n", strings.Join(summary.ActiveRules, ", "))
	}
	_, _ = fmt.Fprintf(stdout, "findings: %d\n", summary.Findings)
}

func renderJSON(writer io.Writer, findings []contractlint.Finding, summary lintSummary, strict bool, pretty bool) error {
	if findings == nil {
		findings = []contractlint.Finding{}
	}
	result := lintResult{
		ResultKind: resultKindLint,
		OK:         len(findings) == 0,
		Summary:    summary,
		Findings:   findings,
		Strict:     strict,
	}
	encoder := json.NewEncoder(writer)
	if pretty {
		encoder.SetIndent(jsonIndentPrefix, jsonIndentValue)
	}
	return encoder.Encode(result)
}
