package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nucleuskit/contract/inspect"
)

const (
	targetContractGen = "contract/gen"
	targetHTTPGen     = "internal/adapter/http/gen"
)

func writeBytesFile(dir string, rel string, data []byte) (string, error) {
	clean, err := cleanServiceRelPath(rel)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, filepath.FromSlash(clean))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return clean, nil
}

func cleanServiceRelPath(rel string) (string, error) {
	if filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") {
		return "", fmt.Errorf("generated path must be relative: %s", rel)
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("generated path escapes service directory: %s", rel)
	}
	return clean, nil
}

func writeFreshnessMarker(dir string, target string, hash string) (string, error) {
	if hash == "" {
		return "", nil
	}
	markerRel := filepath.ToSlash(filepath.Join(filepath.FromSlash(target), inspect.FreshnessMarker))
	return writeBytesFile(dir, markerRel, []byte(hash+"\n"))
}

func relativeFiles(dir string, files []string) []string {
	relative := make([]string, 0, len(files))
	for _, file := range files {
		rel, err := filepath.Rel(dir, file)
		if err == nil && !strings.HasPrefix(rel, "..") && rel != "." {
			relative = append(relative, filepath.ToSlash(rel))
			continue
		}
		relative = append(relative, filepath.ToSlash(file))
	}
	sort.Strings(relative)
	return relative
}

func appendUnique(values []string, next ...string) []string {
	seen := make(map[string]struct{}, len(values)+len(next))
	result := make([]string, 0, len(values)+len(next))
	for _, value := range append(values, next...) {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func contractDocsPath() string {
	return filepath.ToSlash(filepath.Join(targetContractGen, "contract.md"))
}

func typeScriptSchemaPath() string {
	return filepath.ToSlash(filepath.Join(targetContractGen, "types.ts"))
}

func clientOutputPath(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "typescript":
		return filepath.ToSlash(filepath.Join("sdk", "typescript", "client.ts"))
	case "dart":
		return filepath.ToSlash(filepath.Join("sdk", "dart", "client.dart"))
	case "java":
		return filepath.ToSlash(filepath.Join("sdk", "java", "NucleusClient.java"))
	case "kotlin":
		return filepath.ToSlash(filepath.Join("sdk", "kotlin", "NucleusClient.kt"))
	default:
		return filepath.ToSlash(filepath.Join("sdk", strings.ToLower(strings.TrimSpace(language)), "client.txt"))
	}
}

func clientTarget(language string) string {
	return filepath.ToSlash(filepath.Join("sdk", strings.ToLower(strings.TrimSpace(language))))
}
