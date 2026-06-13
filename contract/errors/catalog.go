package errors

import (
	"os"
	"path/filepath"
	"sort"

	"go.yaml.in/yaml/v3"
)

type Code struct {
	Code       int    `yaml:"code" json:"code"`
	Message    string `yaml:"message" json:"message"`
	HTTPStatus int    `yaml:"http_status" json:"http_status"`
}

type catalog struct {
	Errors []Code `yaml:"errors"`
}

func Load(dir string) ([]Code, error) {
	path := filepath.Join(dir, "api", "errors.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var catalog catalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return nil, err
	}
	sort.Slice(catalog.Errors, func(i, j int) bool {
		return catalog.Errors[i].Code < catalog.Errors[j].Code
	})
	return catalog.Errors, nil
}
