package describe

import (
	"encoding/json"

	"github.com/nucleuskit/contract/inspect"
)

const defaultSchemaVersion = "1.1"

// OutputOptions controls the describe JSON payload.
type OutputOptions struct {
	Dir            string
	SchemaOverride string
	IncludeFlow    bool
}

// BuildOutput describes a service directory and adds CLI verification metadata.
func BuildOutput(opts OutputOptions) (map[string]any, error) {
	description, err := inspect.Describe(opts.Dir)
	if err != nil {
		return nil, err
	}
	if opts.IncludeFlow {
		flowGraph, err := inspect.BuildFlowGraphFromDir(opts.Dir)
		if err != nil {
			return nil, err
		}
		description.FlowGraph = &flowGraph
	}
	data, err := descriptionAsMap(description)
	if err != nil {
		return nil, err
	}
	data["schema_version"] = schemaVersion(opts.SchemaOverride)
	data["verification"] = verificationContract()
	return data, nil
}

// schemaVersion returns the schema version to use for the describe output.
func schemaVersion(override string) string {
	if override != "" {
		return override
	}
	return defaultSchemaVersion
}

// descriptionAsMap converts a Description to a map[string]any.
func descriptionAsMap(description inspect.Description) (map[string]any, error) {
	data, err := json.Marshal(description)
	if err != nil {
		return nil, err
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	return value, nil
}
