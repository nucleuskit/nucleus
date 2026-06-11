package inspect

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nucleuskit/contract/openapi"
)

func collectRuntimeHTTPHandlers(dir string, routes []openapi.Route) []sourceRouteHandler {
	operations := map[string]string{}
	for _, route := range routes {
		operations[route.Method+" "+route.Path] = route.OperationID
	}
	var handlers []sourceRouteHandler
	_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			if entry != nil && entry.IsDir() && shouldSkipSourceDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
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
		generatedHTTPAliases := generatedHTTPAdapterAliases(file)
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			methodName := selectorName(call.Fun)
			if methodName == "RegisterRoutes" && generatedHTTPRegisterRoutes(call.Fun, generatedHTTPAliases) && generatedHTTPRoutesFresh(dir) {
				position := fileSet.Position(call.Pos())
				source := relSlash + ":" + strconv.Itoa(position.Line)
				for _, route := range routes {
					handlers = append(handlers, sourceRouteHandler{
						Method:      route.Method,
						Path:        route.Path,
						OperationID: route.OperationID,
						Name:        "generated route binder dispatch",
						Source:      source,
					})
				}
				return true
			}
			if methodName != "Handle" && methodName != "RegisterWellKnown" {
				return true
			}
			handler, ok := routeHandlerFromCall(fileSet, relSlash, call, methodName, operations)
			if ok {
				handlers = append(handlers, handler)
			}
			return true
		})
		return nil
	})
	return handlers
}

func generatedHTTPAdapterAliases(file *ast.File) map[string]bool {
	aliases := map[string]bool{}
	for _, spec := range file.Imports {
		importPath := strings.Trim(spec.Path.Value, `"`)
		if !strings.HasSuffix(importPath, "/internal/adapter/http/gen") {
			continue
		}
		name := filepath.Base(importPath)
		if spec.Name != nil && spec.Name.Name != "_" && spec.Name.Name != "." {
			name = spec.Name.Name
		}
		aliases[name] = true
	}
	return aliases
}

func generatedHTTPRegisterRoutes(expr ast.Expr, aliases map[string]bool) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "RegisterRoutes" {
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	return ok && aliases[ident.Name]
}

func generatedHTTPRoutesFresh(dir string) bool {
	sourceHash, err := ContractSourceHash(dir)
	if err != nil || sourceHash == "" {
		return false
	}
	targetHash, err := ReadGeneratedHash(dir, "internal/adapter/http/gen")
	return err == nil && targetHash == sourceHash
}

func routeHandlerFromCall(fileSet *token.FileSet, relPath string, call *ast.CallExpr, methodName string, operations map[string]string) (sourceRouteHandler, bool) {
	switch methodName {
	case "Handle":
		if len(call.Args) < 3 {
			return sourceRouteHandler{}, false
		}
		method, ok := httpMethodLiteral(call.Args[0])
		if !ok {
			return sourceRouteHandler{}, false
		}
		path, ok := stringLiteral(call.Args[1])
		if !ok {
			return sourceRouteHandler{}, false
		}
		function, ok := call.Args[2].(*ast.FuncLit)
		if !ok {
			return sourceRouteHandler{}, false
		}
		position := fileSet.Position(function.Pos())
		operationID := operations[method+" "+path]
		return sourceRouteHandler{
			Method:      method,
			Path:        path,
			OperationID: operationID,
			Name:        method + " " + path + " handler",
			Source:      relPath + ":" + strconv.Itoa(position.Line),
			UsesLog:     funcLitUsesLog(function),
		}, true
	case "RegisterWellKnown":
		if len(call.Args) < 1 {
			return sourceRouteHandler{}, false
		}
		function, ok := call.Args[0].(*ast.FuncLit)
		if !ok {
			return sourceRouteHandler{}, false
		}
		position := fileSet.Position(function.Pos())
		const method = "GET"
		const path = "/.well-known/nucleus.json"
		return sourceRouteHandler{
			Method:      method,
			Path:        path,
			OperationID: operations[method+" "+path],
			Name:        method + " " + path + " handler",
			Source:      relPath + ":" + strconv.Itoa(position.Line),
			UsesLog:     funcLitUsesLog(function),
		}, true
	default:
		return sourceRouteHandler{}, false
	}
}
