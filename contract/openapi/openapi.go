package openapi

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Endpoint 端点
type Endpoint struct {
	Method      string `json:"method"`                 // 请求方法
	Path        string `json:"path"`                   // 路径
	OperationID string `json:"operation_id,omitempty"` // 操作 ID
}

// document 文档
type document struct {
	OpenAPI string                  `yaml:"openapi"` // OpenAPI 版本
	Paths   map[string]endpointPath `yaml:"paths"`   // 路径
}

// endpointPath 端点路径
type endpointPath struct {
	Get     *operation `yaml:"get"`     // GET 请求
	Post    *operation `yaml:"post"`    // POST 请求
	Put     *operation `yaml:"put"`     // PUT 请求
	Patch   *operation `yaml:"patch"`   // PATCH 请求
	Delete  *operation `yaml:"delete"`  // DELETE 请求
	Head    *operation `yaml:"head"`    // HEAD 请求
	Options *operation `yaml:"options"` // OPTIONS 请求
}
type operation struct {
	OperationID string `yaml:"operationId"`
}

// LoadEndpoints 加载端点
func LoadEndpoints(dir string) ([]Endpoint, error) {
	path := filepath.Join(dir, "api", "openapi.yaml")
	// 读取指定路径的openapi.yaml
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var doc document
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	var endpoints []Endpoint
	for route, pathItem := range doc.Paths {
		for method, op := range pathItem.operations() {
			endpoints = append(endpoints, Endpoint{
				Method:      strings.ToUpper(method),
				Path:        route,
				OperationID: op.OperationID,
			})
		}
	}
	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Path == endpoints[j].Path {
			return endpoints[i].Method < endpoints[j].Method
		}
		return endpoints[i].Path < endpoints[j].Path
	})
	return endpoints, nil
}

// operations 获取所有操作
func (path endpointPath) operations() map[string]operation {
	candidates := map[string]*operation{
		"GET":     path.Get,
		"POST":    path.Post,
		"PUT":     path.Put,
		"PATCH":   path.Patch,
		"DELETE":  path.Delete,
		"HEAD":    path.Head,
		"OPTIONS": path.Options,
	}
	operations := make(map[string]operation, len(candidates))
	for method, op := range candidates {
		if op != nil {
			operations[method] = *op
		}
	}
	return operations
}
