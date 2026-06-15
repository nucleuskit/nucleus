package plan

import (
	"strings"

	"github.com/nucleuskit/contract/inspect"
)

func suggestedEdits(taskType string, description inspect.Description, requestedCapabilities []string) ([]string, []string) {
	edits := uniqueStrings([]string{})
	switch taskType {
	case taskTypeGRPCService:
		if len(description.GRPCServices) == 0 {
			edits = append(edits, "api/proto/*.proto")
		} else {
			for _, service := range description.GRPCServices {
				edits = append(edits, service.Source)
			}
		}
		edits = append(edits, "api/errors.yaml", "internal/domain/**", "internal/adapter/grpc/**")
	case taskTypeCapability:
		edits = append(edits, "nucleus.yaml", "go.mod", "go.sum", "internal/app/**", "internal/config/**", "internal/adapter/store/**", "configs/**", "deploy/**", "docs/**")
	case taskTypeErrorCatalog:
		edits = append(edits, "api/errors.yaml", "internal/domain/**")
	case taskTypeHTTPEndpoint:
		edits = append(edits, "api/openapi.yaml", "api/errors.yaml", "internal/domain/**", "internal/adapter/http/**")
	default:
		edits = append(edits, description.EditSurfaces.Allowed...)
	}
	for _, capability := range requestedCapabilities {
		edits = append(edits, "nucleus.yaml")
		if capability == "sql" || capability == "mongo" || capability == "redis" || capability == "kv" || capability == "store" {
			edits = append(edits, "internal/adapter/store/**", "internal/config/**", "configs/**", "deploy/**", "docs/**", "go.mod", "go.sum")
		}
	}
	return filterSuggestedEdits(taskType, uniqueStrings(edits), description.EditSurfaces)
}

func filterSuggestedEdits(taskType string, desired []string, surfaces inspect.EditSurfaces) ([]string, []string) {
	candidates := append([]string{}, desired...)
	candidates = append(candidates, implementationSurfaces(taskType, surfaces.Allowed)...)
	candidates = uniqueStrings(candidates)

	var allowed []string
	var blocked []string
	for _, path := range candidates {
		switch classifyPlanSurface(path, surfaces) {
		case "allowed":
			allowed = append(allowed, path)
		case "readonly", "forbidden", "manual":
			blocked = append(blocked, path)
		}
	}
	return uniqueStrings(allowed), uniqueStrings(blocked)
}

func implementationSurfaces(taskType string, allowed []string) []string {
	var prefixes []string
	switch taskType {
	case taskTypeHTTPEndpoint:
		prefixes = []string{"cmd/", "internal/domain/", "internal/adapter/http/", "api/"}
	case taskTypeGRPCService:
		prefixes = []string{"cmd/", "internal/domain/", "internal/adapter/grpc/", "api/"}
	case taskTypeCapability:
		prefixes = []string{"nucleus.yaml", "go.mod", "go.sum", "internal/app/", "internal/config/", "internal/adapter/store/", "configs/", "deploy/", "docs/", "test/", "Makefile"}
	case taskTypeErrorCatalog:
		prefixes = []string{"api/", "internal/domain/", "cmd/"}
	default:
		return nil
	}
	var matches []string
	for _, pattern := range allowed {
		normalized := normalizePlanPattern(pattern)
		for _, prefix := range prefixes {
			if normalized == prefix || strings.HasPrefix(normalized, prefix) || strings.HasPrefix(prefix, normalized) {
				matches = append(matches, pattern)
				break
			}
		}
	}
	return matches
}

func classifyPlanSurface(path string, surfaces inspect.EditSurfaces) string {
	switch {
	case matchesAnyPlanSurface(path, surfaces.Forbidden):
		return "forbidden"
	case matchesAnyPlanSurface(path, surfaces.Readonly):
		return "readonly"
	case matchesAnyPlanSurface(path, surfaces.Allowed):
		return "allowed"
	default:
		return "manual"
	}
}

func matchesAnyPlanSurface(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if planSurfaceMatch(path, pattern) {
			return true
		}
	}
	return false
}

func planSurfaceMatch(path string, pattern string) bool {
	pattern = strings.TrimPrefix(strings.ReplaceAll(pattern, "\\", "/"), "./")
	path = strings.TrimPrefix(strings.ReplaceAll(path, "\\", "/"), "./")
	if pattern == path || pattern == "**" {
		return true
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*") + "/"
		rest := strings.TrimPrefix(path, prefix)
		return strings.HasPrefix(path, prefix) && rest != "" && !strings.Contains(rest, "/")
	}
	return false
}

func normalizePlanPattern(pattern string) string {
	pattern = strings.TrimPrefix(strings.ReplaceAll(pattern, "\\", "/"), "./")
	pattern = strings.TrimSuffix(pattern, "**")
	pattern = strings.TrimSuffix(pattern, "*")
	return strings.TrimSuffix(pattern, "/") + "/"
}
