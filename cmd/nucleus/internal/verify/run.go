package verify

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
	contractlint "github.com/nucleuskit/contract/lint"
	"github.com/nucleuskit/contract/validation"
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

	findings := contractlint.Run(dir, true)
	steps = append(steps, verifyStep{
		Phase:   phaseLint,
		Command: commandLintStrict,
		OK:      len(findings) == 0,
	})

	freshnessStep := runGeneratedFreshness(dir)
	steps = append(steps, freshnessStep)
	if len(findings) > 0 || !freshnessStep.OK {
		result := buildResult(steps, diagnostics, findings)
		return result, ErrVerifyFailed
	}

	tidyStep := runTidyCommand(dir)
	steps = append(steps, tidyStep)
	if !tidyStep.OK {
		result := buildResult(steps, diagnostics, findings)
		return result, ErrVerifyFailed
	}

	for _, command := range []struct {
		phase string
		args  []string
	}{
		{phaseImport, []string{"list", "./..."}},
		{phaseBuild, []string{"test", "./...", "-run", "^$"}},
		{phaseTest, []string{"test", "./..."}},
	} {
		step := runGoCommand(dir, command.phase, command.args)
		steps = append(steps, step)
		if !step.OK {
			result := buildResult(steps, diagnostics, findings)
			return result, ErrVerifyFailed
		}
	}

	return buildResult(steps, diagnostics, findings), nil
}

func runGoCommand(dir string, phase string, args []string) verifyStep {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	step := verifyStep{
		Phase:   phase,
		Command: "go " + strings.Join(args, " "),
		OK:      err == nil,
		Output:  strings.TrimSpace(output.String()),
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
