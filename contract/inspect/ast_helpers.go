package inspect

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// isGeneratedSourcePath returns true if the path is generated.
func isGeneratedSourcePath(path string) bool {
	return strings.HasPrefix(path, contractGeneratedTarget+"/") ||
		strings.HasPrefix(path, generatedHTTPAdapterTarget+"/") ||
		strings.HasPrefix(path, generatedGRPCAdapterTarget+"/") ||
		strings.Contains(path, "/"+generatedHTTPAdapterTarget+"/") ||
		strings.Contains(path, "/"+generatedGRPCAdapterTarget+"/")
}

// selectorName returns the name of the selector expression.
func selectorName(expr ast.Expr) string {
	selected, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	return selected.Sel.Name
}

// httpMethodLiteral returns the HTTP method literal value and true if the expression is a literal.
func httpMethodLiteral(expr ast.Expr) (string, bool) {
	if value, ok := stringLiteral(expr); ok {
		return strings.ToUpper(value), true
	}
	selected, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	if ident, ok := selected.X.(*ast.Ident); !ok || ident.Name != identifierHTTP {
		return "", false
	}
	switch selected.Sel.Name {
	case selectorHTTPMethodGet:
		return httpMethodGet, true
	case selectorHTTPMethodPost:
		return httpMethodPost, true
	case selectorHTTPMethodPut:
		return httpMethodPut, true
	case selectorHTTPMethodPatch:
		return httpMethodPatch, true
	case selectorHTTPMethodDelete:
		return httpMethodDelete, true
	default:
		return "", false
	}
}

// stringLiteral returns the string literal value and true if the expression is a literal.
func stringLiteral(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return value, true
}

// funcLitUsesLog returns true if the function literal uses log.
func funcLitUsesLog(fn *ast.FuncLit) bool {
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
		if !ok {
			return true
		}
		receiver, ok := selected.X.(*ast.Ident)
		if !ok || !strings.Contains(strings.ToLower(receiver.Name), capabilityLog) {
			return true
		}
		switch selected.Sel.Name {
		case selectorLogDebug, selectorLogInfo, selectorLogWarn, selectorLogError:
			found = true
			return false
		default:
			return true
		}
	})
	return found
}

// shouldSkipSourceDir returns true if the directory should be skipped.
func shouldSkipSourceDir(name string) bool {
	switch name {
	case skipDirGit, skipDirGitNexus, skipDirVendor, skipDirNodeModules:
		return true
	default:
		return false
	}
}

// fileImports returns true if the file imports the given import path.
func fileImports(file *ast.File, importPath string) bool {
	quoted := `"` + importPath + `"`
	for _, spec := range file.Imports {
		if spec.Path != nil && spec.Path.Value == quoted {
			return true
		}
	}
	return false
}

// callName returns the name of the call expression.
func callName(expr ast.Expr) string {
	switch value := expr.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		return value.Sel.Name
	default:
		return ""
	}
}

// exportedOperationName returns the exported operation name.
func exportedOperationName(operationID string) string {
	if operationID == "" {
		return ""
	}
	parts := strings.FieldsFunc(operationID, func(r rune) bool {
		return r == '_' || r == '-' || r == '.'
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "")
}

// paramTarget returns the target of the parameter.
func paramTarget(handler sourceFunction, name string) string {
	if name == "" || name == flowRequestBodyName {
		return ""
	}
	for _, param := range handler.Params {
		if param == name {
			return handler.Name + "." + param
		}
	}
	return ""
}
