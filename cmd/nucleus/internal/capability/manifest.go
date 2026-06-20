package capability

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nucleuskit/contract/diagnostic"
	"github.com/nucleuskit/nucleus/cmd/nucleus/internal/capcatalog"
	"go.yaml.in/yaml/v3"
)

func updatedManifest(path string, spec capcatalog.Spec, provider string) ([]byte, diagnostic.Diagnostics) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errorDiagnostic("capability.manifest_unavailable", "nucleus.yaml is required and must be readable")
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, errorDiagnostic("capability.manifest_parse_failed", fmt.Sprintf("parse nucleus.yaml: %v", err))
	}
	root := documentRoot(&doc)
	if root == nil || root.Kind != yaml.MappingNode {
		return nil, errorDiagnostic("capability.manifest_invalid", "nucleus.yaml must be a YAML mapping")
	}
	ensureCapability(root, spec.Name)
	ensureProvider(root, spec, provider)
	ensureAllowedChanges(root, allowedChangesFor(spec, provider))
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(root); err != nil {
		return nil, errorDiagnostic("capability.manifest_encode_failed", err.Error())
	}
	if err := encoder.Close(); err != nil {
		return nil, errorDiagnostic("capability.manifest_encode_failed", err.Error())
	}
	return buffer.Bytes(), nil
}

func documentRoot(doc *yaml.Node) *yaml.Node {
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return doc.Content[0]
	}
	return doc
}

func ensureCapability(root *yaml.Node, capability string) {
	capabilities := ensureMappingValue(root, "capabilities", yaml.SequenceNode)
	for _, item := range capabilities.Content {
		if item.Value == capability {
			return
		}
	}
	capabilities.Content = append(capabilities.Content, scalarNode(capability))
}

func ensureProvider(root *yaml.Node, spec capcatalog.Spec, provider string) {
	nucleus := ensureMappingValue(root, "nucleus", yaml.MappingNode)
	providers := ensureMappingValue(nucleus, "providers", yaml.MappingNode)
	capabilityConfig := ensureMappingValue(providers, spec.Name, yaml.MappingNode)
	setScalarValue(capabilityConfig, "provider", provider)
	if spec.DSNEnv != "" {
		setScalarValue(capabilityConfig, "dsn_env", spec.DSNEnv)
	}
	if spec.Name == "sql" && provider == "postgres" {
		setScalarValue(capabilityConfig, "driver", "postgres")
	}
}

func ensureAllowedChanges(root *yaml.Node, items []string) {
	ai := ensureMappingValue(root, "ai", yaml.MappingNode)
	allowed := ensureMappingValue(ai, "allowed_changes", yaml.SequenceNode)
	existing := map[string]struct{}{}
	for _, item := range allowed.Content {
		existing[item.Value] = struct{}{}
	}
	for _, item := range items {
		if _, ok := existing[item]; ok {
			continue
		}
		allowed.Content = append(allowed.Content, scalarNode(item))
		existing[item] = struct{}{}
	}
}

func allowedChangesFor(spec capcatalog.Spec, provider string) []string {
	changes := []string{
		"nucleus.yaml",
		filepath.ToSlash(filepath.Join("internal", "component", "**")),
		filepath.ToSlash(filepath.Join("internal", "app", "**")),
		filepath.ToSlash(filepath.Join("docs", "**")),
	}
	if spec.Name == "sql" && provider == "postgres" {
		changes = append(changes,
			"go.mod",
			"go.sum",
			filepath.ToSlash(filepath.Join("internal", "adapter", "store", "**")),
			filepath.ToSlash(filepath.Join("deploy", "**")),
		)
	}
	return changes
}

func ensureMappingValue(root *yaml.Node, key string, kind yaml.Kind) *yaml.Node {
	for index := 0; index+1 < len(root.Content); index += 2 {
		if root.Content[index].Value == key {
			value := root.Content[index+1]
			if value.Kind != kind {
				value.Kind = kind
				value.Tag = tagForKind(kind)
				value.Value = ""
				value.Content = nil
			}
			return value
		}
	}
	value := &yaml.Node{Kind: kind, Tag: tagForKind(kind)}
	root.Content = append(root.Content, scalarNode(key), value)
	return value
}

func setScalarValue(root *yaml.Node, key string, value string) {
	for index := 0; index+1 < len(root.Content); index += 2 {
		if root.Content[index].Value == key {
			root.Content[index+1] = scalarNode(value)
			return
		}
	}
	root.Content = append(root.Content, scalarNode(key), scalarNode(value))
}

func scalarNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

func tagForKind(kind yaml.Kind) string {
	switch kind {
	case yaml.SequenceNode:
		return "!!seq"
	case yaml.MappingNode:
		return "!!map"
	default:
		return "!!str"
	}
}
