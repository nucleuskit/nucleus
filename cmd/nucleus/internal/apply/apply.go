package apply

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nucleuskit/contract/inspect"
)

func BuildEvidence(dir string, planPath string) (map[string]any, error) {
	plan, err := readJSONObject(planPath)
	if err != nil {
		return nil, err
	}
	description, err := inspect.Describe(dir)
	if err != nil {
		return nil, err
	}
	edits := anyMapSlice(plan["edits"])
	steps := make([]map[string]any, 0, len(edits))
	pass := true
	for index, edit := range edits {
		path, _ := edit["path"].(string)
		surface := classifyEditSurface(path, description.EditSurfaces)
		allowed := surface == "allowed"
		if !allowed {
			pass = false
		}
		steps = append(steps, map[string]any{
			"id":          fmt.Sprintf("apply-dry-run-%d", index+1),
			"kind":        "edit_surface_check",
			"path":        path,
			"surface":     surface,
			"pass":        allowed,
			"exit_code":   boolExitCode(allowed),
			"needs_write": false,
		})
	}
	return map[string]any{
		"schema_version": "evidence.v1",
		"kind":           "nucleus.apply_evidence",
		"mode":           "dry-run",
		"pass":           pass,
		"steps":          steps,
		"diffs":          []string{},
		"rollback_points": []map[string]any{
			{"id": "dry-run", "strategy": "no writes performed", "available": true},
		},
	}, nil
}

func Apply(dir string, planPath string) (map[string]any, error) {
	plan, err := readJSONObject(planPath)
	if err != nil {
		return nil, err
	}
	description, err := inspect.Describe(dir)
	if err != nil {
		return nil, err
	}
	edits := anyMapSlice(plan["edits"])
	commands := anyMapSlice(plan["commands"])
	steps := make([]map[string]any, 0, len(edits)+len(commands))
	pass := true
	for index, edit := range edits {
		path, _ := edit["path"].(string)
		surface := classifyEditSurface(path, description.EditSurfaces)
		allowed := surface == "allowed"
		if _, ok := edit["content"].(string); !ok {
			allowed = false
		}
		if !allowed {
			pass = false
		}
		steps = append(steps, map[string]any{
			"id":          fmt.Sprintf("apply-check-%d", index+1),
			"kind":        "edit_surface_check",
			"path":        path,
			"surface":     surface,
			"pass":        allowed,
			"exit_code":   boolExitCode(allowed),
			"needs_write": true,
		})
	}
	if !pass {
		steps = appendSkippedCommands(steps, commands)
		return applyEvidence("apply", false, steps, []map[string]any{}), nil
	}

	rollbackPoints := make([]map[string]any, 0, len(edits))
	for index, edit := range edits {
		path, _ := edit["path"].(string)
		fullPath, err := resolveEditPath(dir, path)
		if err != nil {
			steps = append(steps, map[string]any{
				"id":        fmt.Sprintf("apply-write-%d", index+1),
				"kind":      "file_write",
				"path":      path,
				"pass":      false,
				"exit_code": 1,
				"error":     err.Error(),
			})
			steps = appendSkippedCommands(steps, commands)
			return applyEvidence("apply", false, steps, rollbackPoints), nil
		}
		rollbackPoint, err := buildRollbackPoint(index, path, fullPath)
		if err != nil {
			steps = append(steps, map[string]any{
				"id":        fmt.Sprintf("apply-write-%d", index+1),
				"kind":      "file_write",
				"path":      path,
				"pass":      false,
				"exit_code": 1,
				"error":     err.Error(),
			})
			steps = appendSkippedCommands(steps, commands)
			return applyEvidence("apply", false, steps, rollbackPoints), nil
		}
		if err := rejectSymlinkPath(dir, path); err != nil {
			steps = append(steps, map[string]any{
				"id":        fmt.Sprintf("apply-write-%d", index+1),
				"kind":      "file_write",
				"path":      path,
				"pass":      false,
				"exit_code": 1,
				"error":     err.Error(),
			})
			steps = appendSkippedCommands(steps, commands)
			return applyEvidence("apply", false, steps, rollbackPoints), nil
		}
		rollbackPoints = append(rollbackPoints, rollbackPoint)
		content, _ := edit["content"].(string)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			steps = append(steps, map[string]any{
				"id":        fmt.Sprintf("apply-write-%d", index+1),
				"kind":      "file_write",
				"path":      path,
				"pass":      false,
				"exit_code": 1,
				"error":     err.Error(),
			})
			steps = appendSkippedCommands(steps, commands)
			return applyEvidence("apply", false, steps, rollbackPoints), nil
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			steps = append(steps, map[string]any{
				"id":        fmt.Sprintf("apply-write-%d", index+1),
				"kind":      "file_write",
				"path":      path,
				"pass":      false,
				"exit_code": 1,
				"error":     err.Error(),
			})
			steps = appendSkippedCommands(steps, commands)
			return applyEvidence("apply", false, steps, rollbackPoints), nil
		}
		steps = append(steps, map[string]any{
			"id":        fmt.Sprintf("apply-write-%d", index+1),
			"kind":      "file_write",
			"path":      path,
			"pass":      true,
			"exit_code": 0,
			"bytes":     len(content),
		})
	}
	steps = appendSkippedCommands(steps, commands)
	return applyEvidence("apply", true, steps, rollbackPoints), nil
}

func readJSONObject(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func anyMapSlice(value any) []map[string]any {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if mapped, ok := item.(map[string]any); ok {
			result = append(result, mapped)
		}
	}
	return result
}

func appendSkippedCommands(steps []map[string]any, commands []map[string]any) []map[string]any {
	for index, command := range commands {
		commandText, _ := command["command"].(string)
		steps = append(steps, map[string]any{
			"id":        fmt.Sprintf("command-skipped-%d", index+1),
			"kind":      "command_skipped",
			"command":   commandText,
			"pass":      true,
			"exit_code": 0,
			"reason":    "apply does not execute shell commands",
		})
	}
	return steps
}

func applyEvidence(mode string, pass bool, steps []map[string]any, rollbackPoints []map[string]any) map[string]any {
	return map[string]any{
		"schema_version":  "evidence.v1",
		"kind":            "nucleus.apply_evidence",
		"mode":            mode,
		"pass":            pass,
		"steps":           steps,
		"diffs":           []string{},
		"rollback_points": rollbackPoints,
	}
}

func resolveEditPath(dir string, path string) (string, error) {
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("absolute edit path is not allowed: %s", path)
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	fullPath := filepath.Join(root, filepath.FromSlash(path))
	rel, err := filepath.Rel(root, fullPath)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("edit path escapes service root: %s", path)
	}
	return fullPath, nil
}

func rejectSymlinkPath(dir string, path string) error {
	root, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	current := root
	for _, part := range strings.Split(clean, string(filepath.Separator)) {
		if part == "." || part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("edit path contains symlink: %s", path)
		}
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func buildRollbackPoint(index int, path string, fullPath string) (map[string]any, error) {
	data, err := os.ReadFile(fullPath)
	if err == nil {
		return map[string]any{
			"id":               fmt.Sprintf("rollback-%d", index+1),
			"path":             path,
			"strategy":         "restore original content",
			"available":        true,
			"existed":          true,
			"original_content": string(data),
		}, nil
	}
	if os.IsNotExist(err) {
		return map[string]any{
			"id":        fmt.Sprintf("rollback-%d", index+1),
			"path":      path,
			"strategy":  "remove created file",
			"available": true,
			"existed":   false,
		}, nil
	}
	return nil, err
}

func classifyEditSurface(path string, surfaces inspect.EditSurfaces) string {
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
	pattern = filepath.ToSlash(strings.TrimPrefix(pattern, "./"))
	path = filepath.ToSlash(strings.TrimPrefix(path, "./"))
	if pattern == path || pattern == "**" {
		return true
	}
	if ok, _ := filepath.Match(pattern, path); ok {
		return true
	}
	if strings.HasSuffix(pattern, "/**") {
		return strings.HasPrefix(path, strings.TrimSuffix(pattern, "/**")+"/") || path == strings.TrimSuffix(pattern, "/**")
	}
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*") + "/"
		return strings.HasPrefix(path, prefix) && !strings.Contains(strings.TrimPrefix(path, prefix), "/")
	}
	return false
}

func boolExitCode(pass bool) int {
	if pass {
		return 0
	}
	return 1
}
