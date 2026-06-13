package inspect

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
)

func collectSourceFunctions(dir string) []sourceFunction {
	var functions []sourceFunction
	_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			if entry != nil && entry.IsDir() && shouldSkipSourceDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, goSourceExtension) || strings.HasSuffix(path, goTestSourceExtension) {
			return nil
		}
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			relPath = path
		}
		relSlash := filepath.ToSlash(relPath)
		if isGeneratedSourcePath(relSlash) {
			return nil
		}
		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			return nil
		}
		importsHTTPClient := fileImports(file, moduleCapRoot+"/httpclient")
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			position := fileSet.Position(fn.Pos())
			functions = append(functions, sourceFunction{
				Name:           fn.Name.Name,
				Source:         relSlash + sourceLineSeparator + strconv.Itoa(position.Line),
				Params:         funcParams(fn),
				Calls:          funcCalls(fn),
				Domain:         strings.Contains(relSlash, domainSourcePathPart),
				UsesHTTPClient: importsHTTPClient && funcUsesSelector(fn, selectorHTTPClientDo),
			})
		}
		return nil
	})
	return functions
}

func funcParams(fn *ast.FuncDecl) []string {
	if fn.Type.Params == nil {
		return nil
	}
	var params []string
	for _, field := range fn.Type.Params.List {
		for _, name := range field.Names {
			params = append(params, name.Name)
		}
	}
	return params
}

func funcCalls(fn *ast.FuncDecl) []string {
	seen := map[string]bool{}
	var calls []string
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := callName(call.Fun)
		if name != "" && !seen[name] {
			seen[name] = true
			calls = append(calls, name)
		}
		return true
	})
	return calls
}

func funcUsesSelector(fn *ast.FuncDecl, selector string) bool {
	found := false
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if found {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selected, ok := call.Fun.(*ast.SelectorExpr)
		if ok && selected.Sel.Name == selector {
			found = true
			return false
		}
		return true
	})
	return found
}
