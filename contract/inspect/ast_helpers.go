package inspect

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

func isGeneratedSourcePath(path string) bool {
	return strings.HasPrefix(path, "contract/gen/") ||
		strings.HasPrefix(path, "internal/adapter/http/gen/") ||
		strings.HasPrefix(path, "internal/adapter/grpc/gen/") ||
		strings.Contains(path, "/internal/adapter/http/gen/") ||
		strings.Contains(path, "/internal/adapter/grpc/gen/")
}

func selectorName(expr ast.Expr) string {
	selected, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	return selected.Sel.Name
}

func httpMethodLiteral(expr ast.Expr) (string, bool) {
	if value, ok := stringLiteral(expr); ok {
		return strings.ToUpper(value), true
	}
	selected, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	if ident, ok := selected.X.(*ast.Ident); !ok || ident.Name != "http" {
		return "", false
	}
	switch selected.Sel.Name {
	case "MethodGet":
		return "GET", true
	case "MethodPost":
		return "POST", true
	case "MethodPut":
		return "PUT", true
	case "MethodPatch":
		return "PATCH", true
	case "MethodDelete":
		return "DELETE", true
	default:
		return "", false
	}
}

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
		if !ok || !strings.Contains(strings.ToLower(receiver.Name), "log") {
			return true
		}
		switch selected.Sel.Name {
		case "Debug", "Info", "Warn", "Error":
			found = true
			return false
		default:
			return true
		}
	})
	return found
}

func shouldSkipSourceDir(name string) bool {
	switch name {
	case ".git", ".gitnexus", "vendor", "node_modules":
		return true
	default:
		return false
	}
}

func fileImports(file *ast.File, importPath string) bool {
	quoted := `"` + importPath + `"`
	for _, spec := range file.Imports {
		if spec.Path != nil && spec.Path.Value == quoted {
			return true
		}
	}
	return false
}

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

func paramTarget(handler sourceFunction, name string) string {
	if name == "" || name == "body.required" {
		return ""
	}
	for _, param := range handler.Params {
		if param == name {
			return handler.Name + "." + param
		}
	}
	return ""
}
