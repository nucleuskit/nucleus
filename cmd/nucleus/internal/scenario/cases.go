package scenario

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// HTTPCase describes one explicit HTTP scenario case.
type HTTPCase struct {
	ID         string            `json:"id"`
	Method     string            `json:"method"`
	Path       string            `json:"path"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       json.RawMessage   `json:"body,omitempty"`
	Assertions []Assertion       `json:"assertions,omitempty"`
}

// Assertion describes one HTTP scenario assertion.
type Assertion struct {
	Type     string `json:"type"`
	Path     string `json:"path,omitempty"`
	Equals   any    `json:"equals,omitempty"`
	Contains string `json:"contains,omitempty"`
}

// LoadHTTPCases loads explicit HTTP scenario cases from a JSON file.
//
// The input may be either a bare array of cases or an object with a top-level
// cases field. Missing case IDs and methods are filled with stable defaults.
func LoadHTTPCases(path string) ([]HTTPCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cases, err := decodeHTTPCases(data)
	if err != nil {
		return nil, err
	}
	for index := range cases {
		if strings.TrimSpace(cases[index].ID) == "" {
			cases[index].ID = fmt.Sprintf("case-%d", index+1)
		}
		if strings.TrimSpace(cases[index].Method) == "" {
			cases[index].Method = http.MethodGet
		}
		if strings.TrimSpace(cases[index].Path) == "" {
			return nil, fmt.Errorf("http case %s missing path", cases[index].ID)
		}
	}
	return cases, nil
}

func decodeHTTPCases(data []byte) ([]HTTPCase, error) {
	var cases []HTTPCase
	if err := json.Unmarshal(data, &cases); err == nil {
		return cases, nil
	}
	var wrapped struct {
		Cases []HTTPCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return nil, err
	}
	if wrapped.Cases == nil {
		return nil, fmt.Errorf("HTTP scenario cases JSON must be an array or an object with cases")
	}
	return wrapped.Cases, nil
}

// RunHTTPCases executes explicit HTTP scenario cases and returns evidence.
func RunHTTPCases(options HTTPRunnerOptions, cases []HTTPCase) (map[string]any, error) {
	runner, err := newHTTPRunner(options)
	if err != nil {
		return nil, err
	}
	maxBodyBytes := options.MaxBodyBytes
	if maxBodyBytes <= 0 {
		maxBodyBytes = defaultHTTPBodyMaxBytes
	}
	pass := true
	steps := make([]map[string]any, 0, len(cases))
	samples := make([]map[string]any, 0, len(cases))
	assertions := make([]map[string]any, 0, len(cases))
	for _, testCase := range cases {
		step, sample, results := runHTTPCase(runner, testCase, maxBodyBytes)
		if stepOK, _ := step["ok"].(bool); !stepOK {
			pass = false
		}
		for _, result := range results {
			if resultPass, _ := result["pass"].(bool); !resultPass {
				pass = false
			}
		}
		steps = append(steps, step)
		samples = append(samples, sample)
		assertions = append(assertions, results...)
	}
	status := "passed"
	if !pass {
		status = "failed"
	}
	return map[string]any{
		"result_kind":       httpEvidenceKind,
		"schema_version":    "evidence.v1",
		"schema_ref":        evidenceSchemaRef,
		"ok":                pass,
		"status":            status,
		"steps":             steps,
		"diagnostics":       []map[string]any{},
		"http_samples":      samples,
		"assertion_results": assertionResults(assertions),
		"redaction_applied": true,
		"redaction_policy": map[string]any{
			"keys": []string{"authorization", "cookie", "set-cookie"},
		},
		"executed_by_package": "cmd/nucleus/internal/scenario",
	}, nil
}

func runHTTPCase(runner httpRunner, testCase HTTPCase, maxBodyBytes int64) (map[string]any, map[string]any, []map[string]any) {
	request, body, err := buildHTTPCaseRequest(runner, testCase)
	if err != nil {
		step := map[string]any{
			"id":     testCase.ID,
			"kind":   "http_case",
			"method": testCase.Method,
			"path":   testCase.Path,
			"status": "failed",
			"ok":     false,
			"reason": err.Error(),
		}
		return step, map[string]any{"id": testCase.ID, "response": map[string]any{"error": err.Error()}}, []map[string]any{failedHTTPAssertion(testCase.ID, err.Error())}
	}
	requestSample := buildRequestSample(request, body, maxBodyBytes)
	response, responseBody, err := runner.do(request, maxBodyBytes)
	if err != nil {
		step := map[string]any{
			"id":     testCase.ID,
			"kind":   "http_case",
			"method": testCase.Method,
			"path":   testCase.Path,
			"status": "failed",
			"ok":     false,
			"reason": err.Error(),
		}
		return step, map[string]any{
			"id":       testCase.ID,
			"request":  requestSample,
			"response": map[string]any{"error": err.Error()},
		}, []map[string]any{failedHTTPAssertion(testCase.ID, err.Error())}
	}
	envelope, hasEnvelope := unwrapEnvelope(responseBody)
	responseSample := buildResponseSample(response, responseBody, maxBodyBytes, envelope, hasEnvelope)
	results := evaluateAssertions(testCase, response.StatusCode, responseBody)
	casePass := true
	for _, result := range results {
		if ok, _ := result["pass"].(bool); !ok {
			casePass = false
		}
	}
	status := "passed"
	if !casePass {
		status = "failed"
	}
	step := map[string]any{
		"id":          testCase.ID,
		"kind":        "http_case",
		"method":      testCase.Method,
		"path":        testCase.Path,
		"status":      status,
		"ok":          casePass,
		"http_status": response.StatusCode,
	}
	addEnvelopeFields(step, envelope, hasEnvelope)
	return step, map[string]any{
		"id":       testCase.ID,
		"method":   testCase.Method,
		"path":     testCase.Path,
		"request":  requestSample,
		"response": responseSample,
	}, results
}

func buildHTTPCaseRequest(runner httpRunner, testCase HTTPCase) (*http.Request, []byte, error) {
	method := strings.ToUpper(strings.TrimSpace(testCase.Method))
	if method == "" {
		method = http.MethodGet
	}
	body := []byte(testCase.Body)
	var request *http.Request
	var err error
	if runner.handler != nil {
		request, err = http.NewRequest(method, testCase.Path, bytes.NewReader(body))
	} else {
		var target string
		target, err = joinBaseURL(runner.baseURL, testCase.Path, "")
		if err != nil {
			return nil, nil, err
		}
		request, err = http.NewRequest(method, target, bytes.NewReader(body))
	}
	if err != nil {
		return nil, nil, err
	}
	for key, value := range testCase.Headers {
		request.Header.Set(key, value)
	}
	if len(body) > 0 && request.Header.Get("Content-Type") == "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return request, body, nil
}

func evaluateAssertions(testCase HTTPCase, statusCode int, body []byte) []map[string]any {
	results := make([]map[string]any, 0, len(testCase.Assertions))
	var decoded any
	_ = json.Unmarshal(body, &decoded)
	for index, assertion := range testCase.Assertions {
		result := map[string]any{
			"id":   fmt.Sprintf("%s.assertion-%d", testCase.ID, index+1),
			"type": assertion.Type,
		}
		pass, actual := evaluateAssertion(assertion, statusCode, body, decoded)
		result["pass"] = pass
		result["actual"] = actual
		if assertion.Equals != nil {
			result["expected"] = assertion.Equals
		}
		if assertion.Contains != "" {
			result["expected"] = assertion.Contains
		}
		results = append(results, result)
	}
	return results
}

func evaluateAssertion(assertion Assertion, statusCode int, body []byte, decoded any) (bool, any) {
	switch assertion.Type {
	case "http_status":
		return valuesEqual(float64(statusCode), assertion.Equals), statusCode
	case "json_path_equals":
		actual, ok := lookupJSONPath(decoded, assertion.Path)
		if !ok {
			return false, "<missing>"
		}
		return valuesEqual(actual, assertion.Equals), actual
	case "body_contains":
		actual := string(body)
		return strings.Contains(actual, assertion.Contains), actual
	default:
		return false, "unsupported assertion type"
	}
}

func lookupJSONPath(value any, path string) (any, bool) {
	current := value
	for _, part := range strings.Split(path, ".") {
		if part == "" {
			continue
		}
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func valuesEqual(left any, right any) bool {
	switch typed := left.(type) {
	case float64:
		if expected, ok := numberValue(right); ok {
			return typed == expected
		}
	case int:
		if expected, ok := numberValue(right); ok {
			return float64(typed) == expected
		}
	}
	return fmt.Sprint(left) == fmt.Sprint(right)
}

func numberValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}
