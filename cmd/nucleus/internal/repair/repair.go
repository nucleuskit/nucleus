package repair

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	contractgen "github.com/nucleuskit/contract/gen"
	"github.com/nucleuskit/contract/inspect"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/verify"
)

type patchCandidate struct {
	File         string
	Find         string
	Replace      string
	Reason       string
	ExpectedHash string
}

func BuildEvidence(dir string, evidencePath string, maxRounds int) (map[string]any, error) {
	evidence, err := readJSONObject(evidencePath)
	if err != nil {
		return nil, err
	}
	if evidenceResultKind(evidence) == "nucleus.verify_result" && hasFailedStep(evidence, "generated_freshness") {
		return regenerateAndVerify(dir, maxRounds, "regenerate_generated_freshness"), nil
	}
	if hasMissingGenerated(evidence) {
		return regenerateAndVerify(dir, maxRounds, "regenerate_missing_generated"), nil
	}
	candidate, ok, reason := extractPatchCandidate(evidence)
	if ok {
		return applyPatchCandidate(dir, evidence, candidate, maxRounds), nil
	}
	if reason != "" {
		return manualRepairEvidence(maxRounds, evidenceResultKind(evidence), reason), nil
	}
	return manualRepairEvidence(maxRounds, evidenceResultKind(evidence), "automatic code repair is not implemented in the safe skeleton"), nil
}

func manualRepairEvidence(maxRounds int, evidenceKind any, reason string) map[string]any {
	rounds := []map[string]any{
		{
			"id":            "repair-1",
			"status":        "unsupported",
			"reason":        reason,
			"evidence_kind": evidenceKind,
		},
	}
	return repairEvidence(false, "needs_manual_action", maxRounds, rounds, nil)
}

func regenerateAndVerify(dir string, maxRounds int, strategy string) map[string]any {
	result, genErr := contractgen.GenerateWithOptions(dir, contractgen.Options{HTTP: true, GRPC: true, Errors: true})
	if genErr != nil {
		rounds := []map[string]any{
			{
				"id":       "repair-1",
				"strategy": strategy,
				"status":   "failed",
				"stderr":   genErr.Error(),
			},
		}
		return repairEvidence(false, "failed", maxRounds, rounds, nil)
	}
	verifyResult := verify.BuildResultForDir(dir)
	verificationPass := verifyResult.OK
	status := "failed"
	if verificationPass {
		status = "repaired"
	}
	rounds := []map[string]any{
		{
			"id":              "repair-1",
			"strategy":        strategy,
			"status":          status,
			"generated_files": result.Files,
			"source_hash":     result.Hash,
		},
	}
	return repairEvidence(verificationPass, status, maxRounds, rounds, map[string]any{
		"verification_pass": verificationPass,
		"verify_result":     verifyResult,
	})
}

func applyPatchCandidate(dir string, evidence map[string]any, candidate patchCandidate, maxRounds int) map[string]any {
	description, err := inspect.Describe(dir)
	if err != nil {
		return manualRepairEvidence(maxRounds, evidenceResultKind(evidence), err.Error())
	}
	surface := classifyRepairSurface(candidate.File, description.EditSurfaces, evidenceAllowedFiles(evidence))
	if surface != "allowed" {
		return manualRepairEvidence(maxRounds, evidenceResultKind(evidence), "patch file is not in allowed edit surfaces")
	}
	fullPath, err := resolveRepairPath(dir, candidate.File)
	if err != nil {
		return manualRepairEvidence(maxRounds, evidenceResultKind(evidence), err.Error())
	}
	if err := rejectSymlinkPath(dir, fullPath, candidate.File); err != nil {
		return manualRepairEvidence(maxRounds, evidenceResultKind(evidence), err.Error())
	}
	original, err := os.ReadFile(fullPath)
	if err != nil {
		return manualRepairEvidence(maxRounds, evidenceResultKind(evidence), err.Error())
	}
	originalHash := sha256String(string(original))
	if candidate.ExpectedHash == "" || !strings.EqualFold(candidate.ExpectedHash, originalHash) {
		return manualRepairEvidence(maxRounds, evidenceResultKind(evidence), "patch expected_hash does not match current file")
	}
	if count := strings.Count(string(original), candidate.Find); count != 1 {
		return manualRepairEvidence(maxRounds, evidenceResultKind(evidence), fmt.Sprintf("patch find must match exactly once, got %d", count))
	}
	updated := strings.Replace(string(original), candidate.Find, candidate.Replace, 1)
	if err := os.WriteFile(fullPath, []byte(updated), 0o644); err != nil {
		return manualRepairEvidence(maxRounds, evidenceResultKind(evidence), err.Error())
	}
	verifyResult := verify.BuildResultForDir(dir)
	verificationPass := verifyResult.OK
	status := "failed"
	if verificationPass {
		status = "repaired"
	}
	rounds := []map[string]any{
		{
			"id":       "repair-1",
			"strategy": "bounded_business_patch",
			"status":   status,
			"file":     candidate.File,
			"reason":   candidate.Reason,
			"rollback_point": map[string]any{
				"id":               "rollback-1",
				"path":             candidate.File,
				"strategy":         "restore original content",
				"available":        true,
				"original_hash":    originalHash,
				"original_content": string(original),
			},
			"verification": map[string]any{
				"ok": verificationPass,
			},
		},
	}
	return repairEvidence(verificationPass, status, maxRounds, rounds, map[string]any{
		"verification_pass": verificationPass,
		"verify_result":     verifyResult,
	})
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

func extractPatchCandidate(evidence map[string]any) (patchCandidate, bool, string) {
	candidates := []patchCandidate{}
	for _, step := range evidenceSteps(evidence) {
		if suggestion, ok := step["repair_suggestion"].(map[string]any); ok {
			candidate, valid := mapPatchCandidate(suggestion)
			if !valid {
				return patchCandidate{}, false, "repair_suggestion must include file, find, replace, reason, and expected_hash"
			}
			candidates = append(candidates, candidate)
		}
	}
	if failure, ok := evidence["failure"].(map[string]any); ok {
		if fix, ok := failure["fix_candidate"].(map[string]any); ok {
			candidate, valid := mapPatchCandidate(fix)
			if !valid {
				return patchCandidate{}, false, "fix_candidate must include file, find, replace, reason, and expected_hash"
			}
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) == 0 {
		return patchCandidate{}, false, ""
	}
	if len(candidates) > 1 {
		return patchCandidate{}, false, "repair evidence contains multiple patch candidates"
	}
	return candidates[0], true, ""
}

func mapPatchCandidate(value map[string]any) (patchCandidate, bool) {
	candidate := patchCandidate{
		File:         stringField(value, "file"),
		Find:         rawStringField(value, "find"),
		Replace:      rawStringField(value, "replace"),
		Reason:       stringField(value, "reason"),
		ExpectedHash: stringField(value, "expected_hash"),
	}
	if candidate.File == "" || candidate.Find == "" || candidate.Reason == "" || candidate.ExpectedHash == "" {
		return patchCandidate{}, false
	}
	return candidate, true
}

func stringField(value map[string]any, key string) string {
	text, _ := value[key].(string)
	return strings.TrimSpace(text)
}

func rawStringField(value map[string]any, key string) string {
	text, _ := value[key].(string)
	return text
}

func evidenceResultKind(evidence map[string]any) string {
	return stringField(evidence, "result_kind")
}

func repairEvidence(ok bool, status string, maxRounds int, rounds []map[string]any, extra map[string]any) map[string]any {
	result := map[string]any{
		"result_kind":    resultKindRepairEvidence,
		"schema_version": schemaVersionEvidence,
		"schema_ref":     schemaRefEvidence,
		"ok":             ok,
		"status":         status,
		"max_rounds":     maxRounds,
		"steps":          repairSteps(rounds),
		"diagnostics":    []map[string]any{},
		"rounds":         rounds,
	}
	for key, value := range extra {
		result[key] = value
	}
	return result
}

func repairSteps(rounds []map[string]any) []map[string]any {
	steps := make([]map[string]any, 0, len(rounds))
	for index, round := range rounds {
		id := stringField(round, "id")
		if id == "" {
			id = fmt.Sprintf("repair-%d", index+1)
		}
		status := repairStepStatus(stringField(round, "status"))
		kind := stringField(round, "strategy")
		if kind == "" {
			kind = "repair_round"
		}
		step := map[string]any{
			"id":     id,
			"kind":   kind,
			"status": status,
			"ok":     status == statusPassed,
		}
		if reason := stringField(round, "reason"); reason != "" {
			step["reason"] = reason
		}
		if stderr := stringField(round, "stderr"); stderr != "" {
			step["error"] = stderr
		}
		if file := stringField(round, "file"); file != "" {
			step["file"] = file
		}
		returnedStatus := stringField(round, "status")
		if returnedStatus != "" && returnedStatus != status {
			step["repair_status"] = returnedStatus
		}
		steps = append(steps, step)
	}
	return steps
}

func repairStepStatus(status string) string {
	switch status {
	case "repaired":
		return statusPassed
	case "failed":
		return statusFailed
	default:
		return statusBlocked
	}
}

func hasFailedStep(evidence map[string]any, id string) bool {
	steps, ok := evidence["steps"].([]any)
	if !ok {
		return false
	}
	for _, stepValue := range steps {
		step, ok := stepValue.(map[string]any)
		if !ok {
			continue
		}
		stepID, _ := step["id"].(string)
		stepOK, _ := step["ok"].(bool)
		if stepID == id && !stepOK {
			return true
		}
	}
	return false
}

func hasMissingGenerated(evidence map[string]any) bool {
	for _, step := range evidenceSteps(evidence) {
		if stepIndicatesMissingGenerated(step) {
			return true
		}
	}
	return false
}

func evidenceSteps(evidence map[string]any) []map[string]any {
	steps, ok := evidence["steps"].([]any)
	if !ok {
		return nil
	}
	result := make([]map[string]any, 0, len(steps))
	for _, stepValue := range steps {
		step, ok := stepValue.(map[string]any)
		if ok {
			result = append(result, step)
		}
	}
	return result
}

func stepIndicatesMissingGenerated(step map[string]any) bool {
	stepOK, _ := step["ok"].(bool)
	if stepOK {
		return false
	}
	for _, key := range []string{"id", "kind"} {
		value, _ := step[key].(string)
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "missing_generated" {
			return true
		}
	}
	rule, _ := step["rule"].(string)
	if strings.EqualFold(strings.TrimSpace(rule), "L010") && generatedPath(step) {
		return true
	}
	for _, key := range []string{"reason", "stderr"} {
		value, _ := step[key].(string)
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "l010" && generatedPath(step) {
			return true
		}
	}
	return false
}

func generatedPath(step map[string]any) bool {
	path, _ := step["path"].(string)
	path = strings.TrimPrefix(strings.TrimSpace(path), "./")
	return strings.HasPrefix(path, "contract/gen/") ||
		strings.HasPrefix(path, "internal/adapter/http/gen/") ||
		strings.HasPrefix(path, "internal/adapter/grpc/gen/") ||
		strings.HasPrefix(path, "sdk/go/")
}

func evidenceAllowedFiles(evidence map[string]any) []string {
	values, ok := evidence["allowed_files"].([]any)
	if !ok {
		return nil
	}
	files := make([]string, 0, len(values))
	for _, value := range values {
		if file, ok := value.(string); ok && strings.TrimSpace(file) != "" {
			files = append(files, normalizeRepairPath(file))
		}
	}
	return files
}

func classifyRepairSurface(path string, surfaces inspect.EditSurfaces, evidenceAllowed []string) string {
	switch {
	case matchesAnySurface(path, surfaces.Forbidden):
		return "forbidden"
	case matchesAnySurface(path, surfaces.Readonly):
		return "readonly"
	case matchesAnySurface(path, surfaces.Allowed):
		return "allowed"
	case matchesExactFile(path, evidenceAllowed):
		return "allowed"
	default:
		return "manual"
	}
}

func matchesExactFile(path string, files []string) bool {
	path = normalizeRepairPath(path)
	for _, file := range files {
		if path == file {
			return true
		}
	}
	return false
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
	pattern = normalizeRepairPath(pattern)
	path = normalizeRepairPath(path)
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

func resolveRepairPath(dir string, path string) (string, error) {
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("absolute repair path is not allowed: %s", path)
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
		return "", fmt.Errorf("repair path escapes service root: %s", path)
	}
	return fullPath, nil
}

func rejectSymlinkPath(dir string, fullPath string, displayPath string) error {
	root, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, fullPath)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("repair path escapes service root: %s", displayPath)
	}
	clean := filepath.Clean(rel)
	current := root
	for _, part := range strings.Split(clean, string(filepath.Separator)) {
		if part == "." || part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("repair path contains symlink: %s", displayPath)
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

func normalizeRepairPath(path string) string {
	return filepath.ToSlash(strings.TrimPrefix(strings.TrimSpace(path), "./"))
}

func sha256String(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
