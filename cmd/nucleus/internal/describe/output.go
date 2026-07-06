package describe

import (
	"encoding/json"

	"github.com/nucleuskit/contract/diagnostic"
	"github.com/nucleuskit/contract/inspect"
)

// OutputOptions controls the describe JSON payload.
type OutputOptions struct {
	Dir         string
	IncludeFlow bool
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
	diagnostics := description.Diagnostics
	if diagnostics == nil {
		diagnostics = diagnostic.Diagnostics{}
	}
	data[outputFieldResultKind] = resultKindDescribe
	data[outputFieldSchemaVersion] = schemaVersionDescribe
	data[outputFieldSchemaRef] = schemaRefDescribe
	data[outputFieldOK] = !diagnostics.Failed()
	data[outputFieldDiagnostics] = diagnostics
	data[outputFieldVerification] = verificationContract()
	return data, nil
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
