package scenario

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	contracterrors "github.com/nucleuskit/contract/errors"
	"github.com/nucleuskit/contract/inspect"
	"github.com/nucleuskit/contract/openapi"
)

// BuildScenarioPlan builds contract-derived scenario suggestions for a service directory.
func BuildScenarioPlan(dir string) (map[string]any, error) {
	openAPIPath := filepath.Join(dir, "api", "openapi.yaml")
	if _, err := os.Stat(openAPIPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("missing OpenAPI contract: api/openapi.yaml")
		}
		return nil, err
	}

	routes, err := openapi.LoadRouteRegistry(dir)
	if err != nil {
		return nil, err
	}
	if len(routes) == 0 {
		return nil, fmt.Errorf("api/openapi.yaml does not declare any endpoints")
	}
	errorCodes, err := contracterrors.Load(dir)
	if err != nil {
		return nil, err
	}
	graph, err := inspect.BuildFlowGraphFromDir(dir)
	if err != nil {
		return nil, err
	}
	shapes, err := openapi.LoadRequestShapes(dir)
	if err != nil {
		return nil, err
	}

	invalidArgument := findInvalidArgument(errorCodes)
	scenarios := make([]map[string]any, 0, len(routes)*2+len(errorCodes))
	for _, route := range routes {
		shape := shapes[openapi.RequestShapeKey(route.Method, route.Path, route.OperationID)]
		scenarios = append(scenarios, buildSuccessScenario(route, shape))
		if inputs := validationInputs(route, graph, shape); len(inputs) > 0 {
			scenarios = append(scenarios, buildInvalidArgumentScenario(route, invalidArgument, inputs))
		}
	}
	for _, code := range errorCodes {
		if code.Code == 0 {
			continue
		}
		scenarios = append(scenarios, map[string]any{
			"kind":        "error_assertion_hint",
			"code":        code.Code,
			"message":     code.Message,
			"http_status": code.HTTPStatus,
			"suggestion":  "assert the Nucleus envelope code/message/http status when this contract error is intentionally triggered",
			"source":      "api/errors.yaml",
		})
	}

	return map[string]any{
		"result_kind":    resultKindScenarioPlan,
		"ok":             true,
		"kind":           planKind,
		"schema_version": scenarioSchemaVersion,
		"schema_ref":     scenarioSchemaRef,
		"summary":        scenarioSummary(scenarios),
		"diagnostics":    []map[string]any{},
		"source": map[string]any{
			"openapi":    "api/openapi.yaml",
			"errors":     "api/errors.yaml",
			"flow_graph": "contract/inspect.BuildFlowGraphFromDir",
		},
		"scenarios": scenarios,
	}, nil
}

// BuildHTTPCaseDrafts builds executable HTTP case drafts from scenario suggestions.
func BuildHTTPCaseDrafts(dir string) ([]HTTPCase, error) {
	plan, err := BuildScenarioPlan(dir)
	if err != nil {
		return nil, err
	}
	scenarios, ok := plan["scenarios"].([]map[string]any)
	if !ok {
		return nil, fmt.Errorf("scenario plan missing scenarios")
	}
	cases := make([]HTTPCase, 0, len(scenarios))
	for _, item := range scenarios {
		switch item["kind"] {
		case "success":
			cases = append(cases, successHTTPCase(item))
		case "invalid_argument":
			if _, ok := item["error_code"]; ok {
				cases = append(cases, invalidArgumentHTTPCase(item))
			}
		}
	}
	return cases, nil
}

func buildHTTPCaseDraftOutput(dir string) (map[string]any, error) {
	cases, err := BuildHTTPCaseDrafts(dir)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"result_kind":    resultKindHTTPCaseDrafts,
		"ok":             true,
		"kind":           httpCaseDraftsKind,
		"schema_version": scenarioSchemaVersion,
		"schema_ref":     scenarioSchemaRef,
		"summary": map[string]any{
			"cases": len(cases),
		},
		"diagnostics": []map[string]any{},
		"cases":       cases,
	}, nil
}

func scenarioSummary(scenarios []map[string]any) map[string]any {
	summary := map[string]any{
		"scenarios": len(scenarios),
	}
	for _, scenario := range scenarios {
		kind, _ := scenario["kind"].(string)
		switch kind {
		case "success":
			incrementSummary(summary, "success")
		case "invalid_argument":
			incrementSummary(summary, "invalid_argument")
		case "invalid_argument_hint":
			incrementSummary(summary, "invalid_argument_hints")
		case "error_assertion_hint":
			incrementSummary(summary, "error_assertion_hints")
		}
	}
	return summary
}

func incrementSummary(summary map[string]any, key string) {
	current, _ := summary[key].(int)
	summary[key] = current + 1
}

func successHTTPCase(item map[string]any) HTTPCase {
	method, _ := item["method"].(string)
	path, _ := item["path"].(string)
	operationID, _ := item["operation_id"].(string)
	testCase := HTTPCase{
		ID:      chooseScenarioID(operationID, "success"),
		Method:  method,
		Path:    sampleCaseURI(path, item),
		Headers: sampleCaseHeaders(item),
		Assertions: []Assertion{
			{Type: "http_status", Equals: 200},
			{Type: "json_path_equals", Path: "code", Equals: 0},
		},
	}
	if body := caseBody(item); len(body) > 0 {
		testCase.Body = body
	}
	return testCase
}

func invalidArgumentHTTPCase(item map[string]any) HTTPCase {
	method, _ := item["method"].(string)
	path, _ := item["path"].(string)
	operationID, _ := item["operation_id"].(string)
	status := item["http_status"]
	code := item["error_code"]
	testCase := HTTPCase{
		ID:      chooseScenarioID(operationID, "invalid-argument"),
		Method:  method,
		Path:    invalidCaseURI(path, item),
		Headers: sampleCaseHeaders(item),
	}
	if status != nil {
		testCase.Assertions = append(testCase.Assertions, Assertion{Type: "http_status", Equals: status})
	}
	if code != nil {
		testCase.Assertions = append(testCase.Assertions, Assertion{Type: "json_path_equals", Path: "code", Equals: code})
	}
	if corruptsBody(item) {
		testCase.Body = json.RawMessage(`{}`)
	}
	return testCase
}

func chooseScenarioID(operationID string, suffix string) string {
	if strings.TrimSpace(operationID) == "" {
		return suffix
	}
	return operationID + "." + suffix
}

func sampleCaseURI(path string, item map[string]any) string {
	query := url.Values{}
	for _, parameter := range scenarioParameters(item) {
		name, _ := parameter["name"].(string)
		in, _ := parameter["in"].(string)
		sample := fmt.Sprint(parameter["sample"])
		if sample == "" || sample == "<nil>" {
			sample = sampleValue(fmt.Sprint(parameter["schema_type"]))
		}
		if strings.EqualFold(in, "path") && name != "" {
			path = strings.ReplaceAll(path, "{"+name+"}", url.PathEscape(sample))
		}
		if strings.EqualFold(in, "query") && name != "" && parameter["required"] == true {
			query.Set(name, sample)
		}
	}
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	return path
}

func invalidCaseURI(path string, item map[string]any) string {
	parameters := scenarioParameters(item)
	for _, parameter := range parameters {
		name, _ := parameter["name"].(string)
		in, _ := parameter["in"].(string)
		if strings.EqualFold(in, "path") && name != "" {
			invalid := invalidValue(fmt.Sprint(parameter["schema_type"]))
			return sampleCaseURI(path, mapWithParameterSample(item, name, in, invalid))
		}
		if strings.EqualFold(in, "query") && name != "" {
			return sampleCaseURI(path, mapWithoutParameter(item, name, in))
		}
	}
	return sampleCaseURI(path, item)
}

func scenarioParameters(item map[string]any) []map[string]any {
	raw, ok := item["parameters"].([]map[string]any)
	if ok {
		return raw
	}
	values, ok := item["parameters"].([]any)
	if !ok {
		return nil
	}
	parameters := make([]map[string]any, 0, len(values))
	for _, value := range values {
		if mapped, ok := value.(map[string]any); ok {
			parameters = append(parameters, mapped)
		}
	}
	return parameters
}

func buildSuccessScenario(route openapi.Route, shape openapi.RequestShape) map[string]any {
	scenario := map[string]any{
		"kind":         "success",
		"method":       route.Method,
		"path":         route.Path,
		"operation_id": route.OperationID,
		"parameters":   scenarioParametersForRoute(route, shape),
		"suggestion":   "exercise the declared endpoint with contract-valid inputs and assert a Nucleus envelope with code 0",
		"source":       "api/openapi.yaml",
	}
	if body, sample, source, contentType := requestBodySample(route, shape); len(body) > 0 {
		scenario["request_body_required"] = true
		scenario["request_body_sample"] = sample
		scenario["request_body_sample_source"] = source
		if contentType != "" {
			scenario["request_body_content_type"] = contentType
		}
	}
	return scenario
}

func buildInvalidArgumentScenario(route openapi.Route, code contracterrors.Code, inputs []map[string]any) map[string]any {
	kind := "invalid_argument"
	suggestion := "omit or corrupt one required contract input and assert the invalid argument envelope"
	if code.Code == 0 {
		kind = "invalid_argument_hint"
		suggestion = "omit or corrupt one required contract input, then map the behavior to a declared contract error before making it executable"
	}
	scenario := map[string]any{
		"kind":         kind,
		"method":       route.Method,
		"path":         route.Path,
		"operation_id": route.OperationID,
		"parameters":   inputs,
		"suggestion":   suggestion,
		"source":       "api/openapi.yaml",
	}
	if code.Code != 0 {
		scenario["error_code"] = code.Code
		scenario["error_message"] = code.Message
		scenario["http_status"] = code.HTTPStatus
		scenario["error_source"] = "api/errors.yaml"
	}
	return scenario
}

func validationInputs(route openapi.Route, graph inspect.FlowGraph, shape openapi.RequestShape) []map[string]any {
	inputs := make([]map[string]any, 0, len(route.Parameters)+1)
	for _, parameter := range scenarioParametersForRoute(route, shape) {
		required, _ := parameter["required"].(bool)
		if !required {
			continue
		}
		input := map[string]any{
			"name":        parameter["name"],
			"in":          parameter["in"],
			"schema_type": parameter["schema_type"],
			"required":    true,
		}
		if sample, exists := parameter["sample"]; exists {
			input["sample"] = sample
		}
		if source, exists := parameter["sample_source"]; exists {
			input["sample_source"] = source
		}
		if sample, exists := parameter["invalid_sample"]; exists {
			input["invalid_sample"] = sample
		}
		if target := flowTarget(graph, route, fmt.Sprint(parameter["in"])+"."+fmt.Sprint(parameter["name"])); target != "" {
			input["target"] = target
		}
		inputs = append(inputs, input)
	}
	if route.RequestBodyRequired || shape.Body != nil && shape.Body.Required {
		inputs = append(inputs, map[string]any{
			"name":     "body",
			"in":       "body",
			"required": true,
		})
	}
	return inputs
}

func scenarioParametersForRoute(route openapi.Route, shape openapi.RequestShape) []map[string]any {
	if shape.Method != "" || shape.OperationID != "" {
		parameters := make([]map[string]any, 0, len(shape.Parameters))
		for _, parameter := range shape.Parameters {
			sample, source := schemaSample(parameter.Schema, parameter.Schema.Type)
			parameters = append(parameters, map[string]any{
				"name":           parameter.Name,
				"in":             parameter.In,
				"schema_type":    parameter.Schema.Type,
				"required":       parameter.Required,
				"sample":         sample,
				"sample_source":  source,
				"invalid_sample": invalidValue(parameter.Schema.Type),
			})
		}
		return parameters
	}
	parameters := make([]map[string]any, 0, len(route.Parameters))
	for _, parameter := range route.Parameters {
		sample := sampleValue(parameter.SchemaType)
		parameters = append(parameters, map[string]any{
			"name":           parameter.Name,
			"in":             parameter.In,
			"schema_type":    parameter.SchemaType,
			"required":       parameter.Required,
			"sample":         sample,
			"sample_source":  "type",
			"invalid_sample": invalidValue(parameter.SchemaType),
		})
	}
	return parameters
}

func schemaSample(schema openapi.Schema, fallbackType string) (any, string) {
	if example, ok := openapi.ExampleForSchema(schema); ok {
		if example.Source == "type" {
			return sampleValue(fallbackType), example.Source
		}
		return example.Value, example.Source
	}
	return sampleValue(fallbackType), "type"
}

func requestBodySample(route openapi.Route, shape openapi.RequestShape) (json.RawMessage, any, string, string) {
	if shape.Body != nil {
		if example, ok := openapi.ExampleForSchema(shape.Body.Schema); ok {
			body, err := json.Marshal(example.Value)
			if err == nil {
				return body, example.Value, example.Source, shape.Body.ContentType
			}
		}
		if shape.Body.Required {
			sample := map[string]any{"scenario": "sample"}
			body, _ := json.Marshal(sample)
			return body, sample, "fallback", shape.Body.ContentType
		}
	}
	if route.RequestBodyRequired {
		sample := map[string]any{"scenario": "sample"}
		body, _ := json.Marshal(sample)
		return body, sample, "fallback", "application/json"
	}
	return nil, nil, "", ""
}

func sampleCaseHeaders(item map[string]any) map[string]string {
	headers := map[string]string{}
	for _, parameter := range scenarioParameters(item) {
		name, _ := parameter["name"].(string)
		in, _ := parameter["in"].(string)
		if !strings.EqualFold(in, "header") || name == "" || isSensitiveHeader(name) {
			continue
		}
		headers[name] = fmt.Sprint(parameter["sample"])
	}
	if len(headers) == 0 {
		return nil
	}
	return headers
}

func caseBody(item map[string]any) json.RawMessage {
	sample, ok := item["request_body_sample"]
	if !ok {
		return nil
	}
	body, err := json.Marshal(sample)
	if err != nil {
		return nil
	}
	return body
}

func corruptsBody(item map[string]any) bool {
	for _, parameter := range scenarioParameters(item) {
		in, _ := parameter["in"].(string)
		if strings.EqualFold(in, "body") {
			return true
		}
	}
	return false
}

func mapWithoutParameter(item map[string]any, name string, in string) map[string]any {
	cloned := cloneMap(item)
	parameters := make([]map[string]any, 0, len(scenarioParameters(item)))
	for _, parameter := range scenarioParameters(item) {
		if parameter["name"] == name && strings.EqualFold(fmt.Sprint(parameter["in"]), in) {
			continue
		}
		parameters = append(parameters, parameter)
	}
	cloned["parameters"] = parameters
	return cloned
}

func mapWithParameterSample(item map[string]any, name string, in string, sample string) map[string]any {
	cloned := cloneMap(item)
	parameters := make([]map[string]any, 0, len(scenarioParameters(item)))
	for _, parameter := range scenarioParameters(item) {
		next := cloneMap(parameter)
		if parameter["name"] == name && strings.EqualFold(fmt.Sprint(parameter["in"]), in) {
			next["sample"] = sample
		}
		parameters = append(parameters, next)
	}
	cloned["parameters"] = parameters
	return cloned
}

func flowTarget(graph inspect.FlowGraph, route openapi.Route, name string) string {
	for _, fact := range graph.Params {
		if fact.OperationID == route.OperationID && fact.Name == name {
			return fact.Target
		}
	}
	return ""
}

func findInvalidArgument(errorCodes []contracterrors.Code) contracterrors.Code {
	for _, code := range errorCodes {
		if code.Code == 2 || strings.EqualFold(code.Message, "invalid argument") {
			return code
		}
	}
	return contracterrors.Code{}
}
