package verify

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/nucleuskit/contract/diagnostic"
	contractlint "github.com/nucleuskit/contract/lint"
	"github.com/nucleuskit/contract/manifest"
	"github.com/nucleuskit/contract/validation"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/decision"
)

func run(config Config) (verifyResult, error) {
	dir := stringValue(config.Dir, defaultDir)
	var steps []verifyStep

	diagnostics := validation.ValidateService(dir)
	steps = append(steps, verifyStep{
		Phase:   phaseValidate,
		Command: commandValidate,
		OK:      !diagnostics.Failed(),
	})
	if diagnostics.Failed() {
		result := buildResult(steps, diagnostics, nil)
		return result, ErrVerifyFailed
	}
	m, err := manifest.Load(dir)
	if err != nil {
		diagnostics = append(diagnostics, diagnostic.Diagnostic{Severity: diagnostic.SeverityError, Code: "verify.manifest_read_failed", Path: "nucleus.yaml", Message: err.Error()})
		result := buildResult(steps, diagnostics, nil)
		return result, ErrVerifyFailed
	}

	findings := contractlint.Run(dir, true)
	steps = append(steps, verifyStep{
		Phase:   phaseLint,
		Command: commandLintStrict,
		OK:      len(findings) == 0,
	})

	decisionStep, decisionDiagnostics := runDecisionValidation(dir)
	steps = append(steps, decisionStep)
	diagnostics = append(diagnostics, decisionDiagnostics...)
	if decisionDiagnostics.Failed() {
		result := buildResult(steps, diagnostics, findings)
		return result, ErrVerifyFailed
	}

	freshnessStep := runGeneratedFreshness(dir)
	steps = append(steps, freshnessStep)
	if len(findings) > 0 || !freshnessStep.OK {
		result := buildResult(steps, diagnostics, findings)
		return result, ErrVerifyFailed
	}

	for index, command := range m.Verify.Commands {
		step := runProjectVerifyCommand(dir, command, index)
		steps = append(steps, step)
		if !step.OK {
			result := buildResult(steps, diagnostics, findings)
			return result, ErrVerifyFailed
		}
	}

	return buildResult(steps, diagnostics, findings), nil
}

// BuildResultForDir runs the full verification pipeline for a service directory
// and returns the rendered verification result even when verification fails.
func BuildResultForDir(dir string) verifyResult {
	result, _ := run(Config{Dir: &dir})
	return result
}

func runProjectVerifyCommand(dir string, command string, index int) verifyStep {
	args, parseErr := splitCommand(command)
	step := verifyStep{
		ID:      fmt.Sprintf("%s_%d", phaseVerifyCommand, index+1),
		Phase:   phaseVerifyCommand,
		Command: sanitizeCommandOutput(command, dir),
		OK:      parseErr == nil && len(args) > 0,
	}
	if !step.OK {
		step.Error = "invalid verify command"
		if parseErr != nil {
			step.Error = parseErr.Error()
		}
		step.ExitCode = 1
		return step
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(projectVerifyTimeoutSeconds)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	step.OK = err == nil
	step.Output = sanitizeCommandOutput(output.String(), dir)
	if ctx.Err() == context.DeadlineExceeded {
		step.OK = false
		step.Error = fmt.Sprintf("command timed out after %d seconds", projectVerifyTimeoutSeconds)
		step.ExitCode = 124
		return step
	}
	if err != nil {
		step.Error = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			step.ExitCode = exitErr.ExitCode()
		} else {
			step.ExitCode = 1
		}
	}
	return step
}

func splitCommand(command string) ([]string, error) {
	var result []string
	var current strings.Builder
	var quote rune
	escaped := false
	for _, r := range command {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
		case r == ' ' || r == '\t' || r == '\n':
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if escaped {
		current.WriteRune('\\')
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote in command")
	}
	if current.Len() > 0 {
		result = append(result, current.String())
	}
	return result, nil
}

func runDecisionValidation(dir string) (verifyStep, diagnostic.Diagnostics) {
	quality := decision.QualityForDir(dir)
	step := verifyStep{
		Phase:           phaseDecision,
		Command:         commandDecisionValidate,
		OK:              !quality.Diagnostics.Failed(),
		DecisionQuality: &quality,
		Output: fmt.Sprintf(
			"decisions: files=%d valid=%d accepted_locked=%d supersedes=%d drift=%d diagnostics=%d errors, %d warnings",
			quality.Files,
			quality.Valid,
			quality.AcceptedLocked,
			quality.Supersedes,
			quality.Drift,
			quality.Errors,
			quality.Warnings,
		),
	}
	return step, quality.Diagnostics
}

func buildResult(steps []verifyStep, diagnostics diagnostic.Diagnostics, findings []contractlint.Finding) verifyResult {
	steps = completeStepMetadata(steps)
	summary := verifySummary{
		Steps:        len(steps),
		Errors:       diagnostics.Count(diagnostic.SeverityError),
		Warnings:     diagnostics.Count(diagnostic.SeverityWarning),
		LintFindings: len(findings),
	}
	ok := !diagnostics.Failed() && len(findings) == 0
	for _, step := range steps {
		if step.OK {
			summary.Passed++
			continue
		}
		summary.Failed++
		ok = false
	}
	return verifyResult{
		ResultKind:    resultKindVerify,
		SchemaVersion: schemaVersion,
		SchemaRef:     schemaRef,
		OK:            ok,
		Summary:       summary,
		Steps:         steps,
		Diagnostics:   diagnostics,
		Findings:      findings,
	}
}

func completeStepMetadata(steps []verifyStep) []verifyStep {
	for index := range steps {
		step := &steps[index]
		if step.ID == "" {
			step.ID = step.Phase
		}
		if step.Kind == "" {
			step.Kind = step.Phase
		}
		if step.Sequence == 0 {
			step.Sequence = index + 1
		}
		if step.WorkingDir == "" {
			step.WorkingDir = commandWorkingDir
		}
		if step.SchemaRef == "" {
			step.SchemaRef = schemaRef
		}
		if step.Produces == "" {
			step.Produces = resultKindVerify
		}
		if step.OK {
			step.Status = statusPassed
			continue
		}
		step.Status = statusFailed
		if step.ExitCode == 0 {
			step.ExitCode = 1
		}
	}
	return steps
}
