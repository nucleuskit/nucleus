package execute

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	defaultTimeoutSeconds = 120
	logSummaryLimit       = 4096
)

type executablePlan struct {
	Commands   []planCommand   `json:"commands"`
	Assertions []planAssertion `json:"assertions"`
	Rollback   []rollbackPoint `json:"rollback"`
}

type planCommand struct {
	ID             string `json:"id"`
	Command        string `json:"command"`
	CWD            string `json:"cwd"`
	WorkingDir     string `json:"working_dir"`
	TimeoutSeconds int    `json:"timeout_seconds"`
	Allowed        *bool  `json:"allowed"`
}

type planAssertion struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Target   string `json:"target"`
	Expected string `json:"expected"`
}

type rollbackPoint struct {
	ID       string `json:"id"`
	Target   string `json:"target"`
	Strategy string `json:"strategy"`
	Required bool   `json:"required"`
}

func ExecutePlanCommands(dir string, planPath string, allowlist []string) (map[string]any, error) {
	plan, err := readExecutablePlan(planPath)
	if err != nil {
		return nil, err
	}
	allowedCommands := allowlistSet(allowlist)
	steps := make([]map[string]any, 0, len(plan.Commands))
	logs := make([]map[string]any, 0, len(plan.Commands))
	exitCodes := make([]map[string]any, 0, len(plan.Commands))

	pass := true
	for index, command := range plan.Commands {
		step, logEntry, exitCode := executePlanCommand(dir, command, index, allowedCommands)
		if stepPass, _ := step["pass"].(bool); !stepPass {
			pass = false
		}
		steps = append(steps, step)
		logs = append(logs, logEntry)
		exitCodes = append(exitCodes, exitCode)
	}

	status := "passed"
	if !pass {
		status = "failed"
	}
	return map[string]any{
		"schema_version":      "evidence.v1",
		"kind":                "nucleus.executor_evidence",
		"pass":                pass,
		"status":              status,
		"steps":               steps,
		"logs":                logs,
		"exit_codes":          exitCodes,
		"assertion_results":   assertionResults(plan.Assertions, steps),
		"rollback_points":     rollbackPoints(plan.Rollback),
		"redaction_applied":   true,
		"executed_by_package": "cmd/nucleus/internal/execute",
	}, nil
}

func readExecutablePlan(path string) (executablePlan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return executablePlan{}, err
	}
	var plan executablePlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return executablePlan{}, err
	}
	return plan, nil
}

func executePlanCommand(dir string, command planCommand, index int, allowlist map[string]bool) (map[string]any, map[string]any, map[string]any) {
	id := command.ID
	if id == "" {
		id = fmt.Sprintf("cmd-%d", index+1)
	}
	step := map[string]any{
		"id":      id,
		"kind":    "command_execution",
		"command": redact(command.Command),
	}
	args, err := splitCommand(command.Command)
	if err != nil || len(args) == 0 {
		step["pass"] = false
		step["status"] = "failed"
		step["exit_code"] = 1
		step["stderr"] = redact(errorString(err, "empty command"))
		return step, logEntry(id, "stderr", step["stderr"].(string)), exitCodeEntry(id, command.Command, 1)
	}
	commandName := filepath.Base(args[0])
	if !allowlist[commandName] {
		step["pass"] = false
		step["status"] = "blocked"
		step["exit_code"] = 126
		step["command_name"] = commandName
		step["stderr"] = "command is not in executor allowlist"
		return step, logEntry(id, "stderr", step["stderr"].(string)), exitCodeEntry(id, command.Command, 126)
	}
	if command.Allowed != nil && !*command.Allowed {
		step["pass"] = false
		step["status"] = "blocked"
		step["exit_code"] = 126
		step["command_name"] = commandName
		step["stderr"] = "plan command is marked as not allowed"
		return step, logEntry(id, "stderr", step["stderr"].(string)), exitCodeEntry(id, command.Command, 126)
	}
	cwd, err := resolveCWD(dir, command)
	if err != nil {
		step["pass"] = false
		step["status"] = "failed"
		step["exit_code"] = 1
		step["command_name"] = commandName
		step["stderr"] = redact(err.Error())
		return step, logEntry(id, "stderr", step["stderr"].(string)), exitCodeEntry(id, command.Command, 1)
	}

	timeout := command.TimeoutSeconds
	if timeout <= 0 {
		timeout = defaultTimeoutSeconds
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	execCommand := exec.CommandContext(ctx, args[0], args[1:]...)
	execCommand.Dir = cwd
	var output bytes.Buffer
	execCommand.Stdout = &output
	execCommand.Stderr = &output
	err = execCommand.Run()
	summary := redact(truncate(output.String()))
	step["cwd"] = cwd
	step["command_name"] = commandName
	step["log_summary"] = summary
	if ctx.Err() == context.DeadlineExceeded {
		step["pass"] = false
		step["status"] = "timeout"
		step["exit_code"] = 124
		step["stderr"] = fmt.Sprintf("command timed out after %d seconds", timeout)
		return step, logEntry(id, "combined", summary), exitCodeEntry(id, command.Command, 124)
	}
	if err != nil {
		exitCode := commandExitCode(err)
		step["pass"] = false
		step["status"] = "failed"
		step["exit_code"] = exitCode
		step["stderr"] = redact(err.Error())
		return step, logEntry(id, "combined", summary), exitCodeEntry(id, command.Command, exitCode)
	}
	step["pass"] = true
	step["status"] = "passed"
	step["exit_code"] = 0
	return step, logEntry(id, "combined", summary), exitCodeEntry(id, command.Command, 0)
}

func allowlistSet(allowlist []string) map[string]bool {
	set := map[string]bool{}
	for _, item := range allowlist {
		item = strings.TrimSpace(item)
		if item != "" {
			set[filepath.Base(item)] = true
		}
	}
	return set
}

func resolveCWD(dir string, command planCommand) (string, error) {
	cwd := command.CWD
	if cwd == "" {
		cwd = command.WorkingDir
	}
	if cwd == "" {
		cwd = "."
	}
	if filepath.IsAbs(cwd) {
		return "", fmt.Errorf("absolute command cwd is not allowed: %s", cwd)
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	fullPath := filepath.Join(root, filepath.FromSlash(cwd))
	rel, err := filepath.Rel(root, fullPath)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("command cwd escapes service root: %s", cwd)
	}
	return fullPath, nil
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

func commandExitCode(err error) int {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return 1
}

func logEntry(id string, stream string, summary string) map[string]any {
	return map[string]any{
		"id":      id,
		"stream":  stream,
		"summary": redact(truncate(summary)),
	}
}

func exitCodeEntry(id string, command string, exitCode int) map[string]any {
	return map[string]any{
		"id":        id,
		"command":   redact(command),
		"exit_code": exitCode,
	}
}

func assertionResults(assertions []planAssertion, steps []map[string]any) []map[string]any {
	results := make([]map[string]any, 0, len(assertions))
	for _, assertion := range assertions {
		pass := false
		for _, step := range steps {
			if step["command"] == assertion.Target && step["exit_code"] == 0 {
				pass = true
				break
			}
		}
		results = append(results, map[string]any{
			"id":       assertion.ID,
			"type":     assertion.Type,
			"target":   assertion.Target,
			"expected": assertion.Expected,
			"pass":     pass,
		})
	}
	return results
}

func rollbackPoints(points []rollbackPoint) []map[string]any {
	result := make([]map[string]any, 0, len(points))
	for _, point := range points {
		result = append(result, map[string]any{
			"id":        point.ID,
			"target":    point.Target,
			"strategy":  point.Strategy,
			"required":  point.Required,
			"available": false,
		})
	}
	return result
}

var sensitiveKeyPattern = regexp.MustCompile(`(?i)\b(token|password|passwd|secret|dsn)\b(\s*[:=]\s*)(?:"[^"]*"|'[^']*'|[^\s,;]+)`)

func redact(value string) string {
	return sensitiveKeyPattern.ReplaceAllString(value, `${1}${2}<redacted>`)
}

func truncate(output string) string {
	output = strings.TrimSpace(output)
	if len(output) <= logSummaryLimit {
		return output
	}
	return output[:logSummaryLimit] + "...<truncated>"
}

func errorString(err error, fallback string) string {
	if err != nil {
		return err.Error()
	}
	return fallback
}
