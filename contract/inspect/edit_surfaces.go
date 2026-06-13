package inspect

import "github.com/nucleuskit/contract/manifest"

func editSurfaces(m manifest.Manifest) EditSurfaces {
	return EditSurfaces{
		Allowed: withEssentialEditSurfaces(choose(m.AI.AllowedChanges, []string{
			"api/**",
			"cmd/**",
			"internal/app/**",
			"internal/config/**",
			"internal/domain/**",
			"internal/usecase/**",
			"internal/component/**",
			"internal/adapter/http/**",
			"internal/adapter/grpc/**",
			"internal/adapter/store/**",
			"internal/adapter/sdk/**",
			"configs/**",
			"deploy/**",
			"docs/**",
			"test/**",
		})),
		Readonly:  choose(m.AI.Readonly, []string{"internal/adapter/http/gen/**", "internal/adapter/grpc/gen/**", "contract/gen/**", "sdk/go/**"}),
		Forbidden: choose(m.AI.Forbidden, []string{"configs/*.local.yaml", "bridge/legacy/**"}),
	}
}

func withEssentialEditSurfaces(values []string) []string {
	return uniqueEditSurfaces(append(values, "nucleus.yaml", "go.mod", "go.sum"))
}

func choose(values []string, fallback []string) []string {
	if len(values) > 0 {
		return values
	}
	return fallback
}

func uniqueEditSurfaces(values []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
