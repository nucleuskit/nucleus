package inspect

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ImportGraph 创建导入图
func ImportGraph(dir string) ([]string, error) {
	seen := map[string]bool{}
	if err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, goSourceExtension) {
			return nil
		}
		if skipPath(path) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			seen[strings.Trim(spec.Path.Value, `"`)] = true
		}
		return nil
	}); err != nil {
		return nil, err
	}

	imports := make([]string, 0, len(seen))
	for importPath := range seen {
		imports = append(imports, importPath)
	}
	sort.Strings(imports)
	return imports, nil
}

// skipPath 跳过路径
func skipPath(path string) bool {
	clean := filepath.ToSlash(path)
	return strings.Contains(clean, skipPathGit) ||
		strings.Contains(clean, skipPathIdea) ||
		strings.Contains(clean, skipPathCursor)
}
