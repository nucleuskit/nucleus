package inspect

import (
	"github.com/nucleuskit/contract/errors"
	"github.com/nucleuskit/contract/manifest"
	"github.com/nucleuskit/contract/openapi"
	"github.com/nucleuskit/contract/proto"
)

// Description is the JSON-serializable service metadata emitted by describe.
type Description struct {
	SchemaVersion      string                `json:"schema_version"`
	Service            manifest.Service      `json:"service"`
	Capabilities       []string              `json:"capabilities"`
	Endpoints          []openapi.Endpoint    `json:"endpoints"`
	GRPCServices       []proto.Service       `json:"grpc_services,omitempty"`
	ErrorCodes         []errors.Code         `json:"error_codes"`
	Dependencies       []manifest.Dependency `json:"dependencies"`
	Modules            []string              `json:"modules"`
	ConfigKeys         []ConfigKey           `json:"config_keys,omitempty"`
	Policy             map[string]any        `json:"policy"`
	EditSurfaces       EditSurfaces          `json:"edit_surfaces"`
	GeneratedFreshness []GeneratedFreshness  `json:"generated_freshness"`
	CapabilityGraph    []CapabilityNode      `json:"capability_graph"`
	FlowGraph          *FlowGraph            `json:"flow_graph,omitempty"`
	Verification       Verification          `json:"verification"`
}

// EditSurfaces describes paths that AI-assisted changes may edit, read, or avoid.
type EditSurfaces struct {
	Allowed   []string `json:"allowed"`
	Readonly  []string `json:"readonly"`
	Forbidden []string `json:"forbidden"`
}

// GeneratedFreshness records whether generated targets match current contract sources.
type GeneratedFreshness struct {
	Source     string `json:"source"`
	Target     string `json:"target"`
	SourceHash string `json:"source_hash,omitempty"`
	TargetHash string `json:"target_hash,omitempty"`
	Fresh      bool   `json:"fresh"`
	Reason     string `json:"reason,omitempty"`
}

// ConfigKey describes a configuration key discovered from safe config fixtures.
type ConfigKey struct {
	Key          string `json:"key"`
	Source       string `json:"source"`
	Env          string `json:"env,omitempty"`
	DefaultValue string `json:"default_value,omitempty"`
	Inferred     bool   `json:"inferred"`
}

// CapabilityNode connects declared capabilities to imports and configured providers.
type CapabilityNode struct {
	Capability string   `json:"capability"`
	Declared   bool     `json:"declared"`
	Imported   bool     `json:"imported"`
	Provider   string   `json:"provider,omitempty"`
	Module     string   `json:"module,omitempty"`
	Imports    []string `json:"imports,omitempty"`
}

// FlowGraph is a conservative graph of routes, handlers, capabilities, and errors.
type FlowGraph struct {
	SchemaVersion string     `json:"schema_version"`
	Nodes         []FlowNode `json:"nodes"`
	Edges         []FlowEdge `json:"edges"`
	Params        []FlowFact `json:"params"`
	ContextFields []FlowFact `json:"context_fields"`
	ErrorPaths    []FlowFact `json:"error_paths"`
}

// FlowNode is a node in the describe flow graph.
type FlowNode struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Method      string `json:"method,omitempty"`
	Path        string `json:"path,omitempty"`
	OperationID string `json:"operation_id,omitempty"`
	Source      string `json:"source,omitempty"`
	Inferred    bool   `json:"inferred"`
}

// FlowEdge is a directed relationship between flow graph nodes.
type FlowEdge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Kind     string `json:"kind"`
	Inferred bool   `json:"inferred"`
}

// FlowFact is supporting evidence attached to a flow graph.
type FlowFact struct {
	Name        string `json:"name"`
	Kind        string `json:"kind,omitempty"`
	Source      string `json:"source,omitempty"`
	Target      string `json:"target,omitempty"`
	Route       string `json:"route,omitempty"`
	OperationID string `json:"operation_id,omitempty"`
	Required    bool   `json:"required,omitempty"`
	SchemaType  string `json:"schema_type,omitempty"`
	Code        int    `json:"code,omitempty"`
	HTTPStatus  int    `json:"http_status,omitempty"`
	Confidence  string `json:"confidence,omitempty"`
	Inferred    bool   `json:"inferred"`
}

// Verification lists commands expected to validate described metadata.
type Verification struct {
	Commands []string `json:"commands"`
}
