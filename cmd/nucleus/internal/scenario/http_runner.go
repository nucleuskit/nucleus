package scenario

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"github.com/nucleuskit/contract/openapi"
)

const (
	httpEvidenceKind        = "nucleus.http_scenario_evidence"
	evidenceSchemaRef       = "contract/schema/evidence.schema.json"
	defaultHTTPBodyMaxBytes = 8192
	defaultHTTPTimeout      = 30 * time.Second
)

// RequestHook customizes generated HTTP requests before execution.
type RequestHook func(*http.Request, map[string]any) error

// HTTPRunnerOptions controls generated or explicit HTTP scenario execution.
type HTTPRunnerOptions struct {
	BaseURL      string
	Handler      http.Handler
	Client       *http.Client
	RequestHook  RequestHook
	MaxBodyBytes int64
}

// RunHTTPScenarios executes generated HTTP success scenarios and returns evidence.
func RunHTTPScenarios(dir string, options HTTPRunnerOptions) (map[string]any, error) {
	plan, err := BuildScenarioPlan(dir)
	if err != nil {
		return nil, err
	}
	routes, err := openapi.LoadRouteRegistry(dir)
	if err != nil {
		return nil, err
	}
	shapes, err := openapi.LoadRequestShapes(dir)
	if err != nil {
		return nil, err
	}
	runner, err := newHTTPRunner(options)
	if err != nil {
		return nil, err
	}
	maxBodyBytes := options.MaxBodyBytes
	if maxBodyBytes <= 0 {
		maxBodyBytes = defaultHTTPBodyMaxBytes
	}

	pass := true
	steps := make([]map[string]any, 0, len(routes))
	samples := make([]map[string]any, 0, len(routes))
	assertions := make([]map[string]any, 0, len(routes))
	for index, route := range routes {
		shape := shapes[openapi.RequestShapeKey(route.Method, route.Path, route.OperationID)]
		scenario := successScenarioForRoute(plan, route, shape)
		step, sample, assertion := runHTTPScenario(runner, route, shape, scenario, index, options.RequestHook, maxBodyBytes)
		if stepPass, _ := step["pass"].(bool); !stepPass {
			pass = false
		}
		steps = append(steps, step)
		samples = append(samples, sample)
		assertions = append(assertions, assertion)
	}

	status := "passed"
	if !pass {
		status = "failed"
	}
	return map[string]any{
		"schema_version":    "evidence.v1",
		"schema_ref":        evidenceSchemaRef,
		"kind":              httpEvidenceKind,
		"pass":              pass,
		"status":            status,
		"scenario_plan":     plan,
		"steps":             steps,
		"http_samples":      samples,
		"assertion_results": assertionResults(assertions),
		"redaction_applied": true,
		"redaction_policy": map[string]any{
			"keys": []string{"authorization", "cookie", "set-cookie"},
		},
		"executed_by_package": "cmd/nucleus/internal/scenario",
	}, nil
}

type httpRunner struct {
	baseURL string
	handler http.Handler
	client  *http.Client
}

func newHTTPRunner(options HTTPRunnerOptions) (httpRunner, error) {
	baseURL := strings.TrimSpace(options.BaseURL)
	if options.Handler == nil && baseURL == "" {
		return httpRunner{}, fmt.Errorf("HTTP scenario runner requires a handler or --base-url")
	}
	if baseURL != "" {
		parsed, err := url.Parse(baseURL)
		if err != nil {
			return httpRunner{}, err
		}
		if parsed.Scheme == "" || parsed.Host == "" {
			return httpRunner{}, fmt.Errorf("base URL must include scheme and host")
		}
	}
	client := options.Client
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}
	return httpRunner{baseURL: baseURL, handler: options.Handler, client: client}, nil
}

func runHTTPScenario(runner httpRunner, route openapi.Route, shape openapi.RequestShape, scenario map[string]any, index int, hook RequestHook, maxBodyBytes int64) (map[string]any, map[string]any, map[string]any) {
	id := scenarioID(route, index)
	request, requestBody, err := buildHTTPRequest(runner, route, shape, id)
	if err != nil {
		return failedHTTPBuildStep(id, route, err), failedHTTPSample(id, route, err), failedHTTPAssertion(id, err.Error())
	}
	if hook != nil {
		if err := hook(request, scenario); err != nil {
			return failedHTTPBuildStep(id, route, err), requestHookSample(id, route, request, requestBody, err, maxBodyBytes), failedHTTPAssertion(id, err.Error())
		}
	}

	requestSample := buildRequestSample(request, requestBody, maxBodyBytes)
	started := time.Now()
	response, responseBody, err := runner.do(request, maxBodyBytes)
	duration := time.Since(started)
	if err != nil {
		step := failedHTTPBuildStep(id, route, err)
		step["duration_ms"] = durationMilliseconds(duration)
		return step, map[string]any{
			"id":           id,
			"method":       route.Method,
			"path":         route.Path,
			"operation_id": route.OperationID,
			"request":      requestSample,
			"response":     map[string]any{"error": err.Error()},
		}, failedHTTPAssertion(id, err.Error())
	}

	envelope, hasEnvelope := unwrapEnvelope(responseBody)
	pass, reason := httpScenarioPass(response.StatusCode, envelope, hasEnvelope)
	status := "passed"
	if !pass {
		status = "failed"
	}
	step := map[string]any{
		"id":           id,
		"kind":         "http_scenario",
		"method":       route.Method,
		"path":         route.Path,
		"operation_id": route.OperationID,
		"status":       status,
		"pass":         pass,
		"http_status":  response.StatusCode,
		"duration_ms":  durationMilliseconds(duration),
	}
	if reason != "" {
		step["reason"] = reason
	}
	addEnvelopeFields(step, envelope, hasEnvelope)

	responseSample := buildResponseSample(response, responseBody, maxBodyBytes, envelope, hasEnvelope)
	sample := map[string]any{
		"id":           id,
		"method":       route.Method,
		"path":         route.Path,
		"operation_id": route.OperationID,
		"request":      requestSample,
		"response":     responseSample,
	}
	assertion := map[string]any{
		"id":       id + ".nucleus_envelope",
		"type":     "http_nucleus_envelope",
		"target":   route.Method + " " + route.Path,
		"expected": "2xx HTTP status and Nucleus envelope code == 0",
		"actual":   httpAssertionActual(response.StatusCode, envelope, hasEnvelope),
		"pass":     pass,
	}
	return step, sample, assertion
}

func (runner httpRunner) do(request *http.Request, maxBodyBytes int64) (*http.Response, []byte, error) {
	if runner.handler != nil {
		recorder := httptest.NewRecorder()
		runner.handler.ServeHTTP(recorder, request)
		response := recorder.Result()
		body, err := readBody(response.Body, maxBodyBytes)
		if closeErr := response.Body.Close(); err == nil {
			err = closeErr
		}
		return response, body, err
	}
	response, err := runner.client.Do(request)
	if err != nil {
		return nil, nil, err
	}
	body, err := readBody(response.Body, maxBodyBytes)
	if closeErr := response.Body.Close(); err == nil {
		err = closeErr
	}
	return response, body, err
}

func buildHTTPRequest(runner httpRunner, route openapi.Route, shape openapi.RequestShape, id string) (*http.Request, []byte, error) {
	path, rawQuery := sampleRouteURI(route, shape)
	body := sampleRequestBody(route, shape)
	var request *http.Request
	if runner.handler != nil {
		target := path
		if rawQuery != "" {
			target += "?" + rawQuery
		}
		request = httptest.NewRequest(route.Method, target, bytes.NewReader(body))
	} else {
		target, err := joinBaseURL(runner.baseURL, path, rawQuery)
		if err != nil {
			return nil, nil, err
		}
		request, err = http.NewRequest(route.Method, target, bytes.NewReader(body))
		if err != nil {
			return nil, nil, err
		}
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-Request-Id", id)
	applyGeneratedHeaders(request, route, shape)
	if len(body) > 0 {
		contentType := "application/json"
		if shape.Body != nil && shape.Body.ContentType != "" {
			contentType = shape.Body.ContentType
		}
		request.Header.Set("Content-Type", contentType)
	}
	return request, body, nil
}

func applyGeneratedHeaders(request *http.Request, route openapi.Route, shape openapi.RequestShape) {
	for _, parameter := range scenarioParametersForRoute(route, shape) {
		name, _ := parameter["name"].(string)
		if name == "" || !strings.EqualFold(fmt.Sprint(parameter["in"]), "header") || isSensitiveHeader(name) {
			continue
		}
		request.Header.Set(name, fmt.Sprint(parameter["sample"]))
	}
}

func sampleRouteURI(route openapi.Route, shape openapi.RequestShape) (string, string) {
	path := route.Path
	query := url.Values{}
	for _, parameter := range scenarioParametersForRoute(route, shape) {
		name, _ := parameter["name"].(string)
		value := fmt.Sprint(parameter["sample"])
		if value == "" || value == "<nil>" {
			value = sampleValue(fmt.Sprint(parameter["schema_type"]))
		}
		switch strings.ToLower(fmt.Sprint(parameter["in"])) {
		case "path":
			path = strings.ReplaceAll(path, "{"+name+"}", url.PathEscape(value))
		case "query":
			if parameter["required"] == true {
				query.Set(name, value)
			}
		}
	}
	return path, query.Encode()
}

func sampleRequestBody(route openapi.Route, shape openapi.RequestShape) []byte {
	body, _, _, _ := requestBodySample(route, shape)
	return body
}

func sampleValue(schemaType string) string {
	switch strings.ToLower(strings.TrimSpace(schemaType)) {
	case "boolean":
		return "true"
	case "integer":
		return "1"
	case "number":
		return "1.0"
	default:
		return "sample-string"
	}
}

func invalidValue(schemaType string) string {
	switch strings.ToLower(strings.TrimSpace(schemaType)) {
	case "boolean":
		return "not-a-boolean"
	case "integer", "number":
		return "not-a-number"
	default:
		return "invalid-sample"
	}
}

func joinBaseURL(baseURL string, path string, rawQuery string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	caseURL, err := url.Parse(path)
	if err != nil {
		return "", err
	}
	if rawQuery == "" {
		rawQuery = caseURL.RawQuery
	}
	casePath := caseURL.Path
	if casePath == "" {
		casePath = path
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + casePath
	parsed.RawPath = ""
	parsed.RawQuery = rawQuery
	return parsed.String(), nil
}

func successScenarioForRoute(plan map[string]any, route openapi.Route, shape openapi.RequestShape) map[string]any {
	scenarios, ok := plan["scenarios"].([]map[string]any)
	if !ok {
		return buildSuccessScenario(route, shape)
	}
	for _, candidate := range scenarios {
		if candidate["kind"] == "success" && candidate["method"] == route.Method && candidate["path"] == route.Path {
			return cloneMap(candidate)
		}
	}
	return buildSuccessScenario(route, shape)
}

func cloneMap(value map[string]any) map[string]any {
	cloned := make(map[string]any, len(value))
	for key, item := range value {
		cloned[key] = item
	}
	return cloned
}

func scenarioID(route openapi.Route, index int) string {
	if strings.TrimSpace(route.OperationID) != "" {
		return route.OperationID
	}
	return fmt.Sprintf("http-%d", index+1)
}

func buildRequestSample(request *http.Request, body []byte, maxBodyBytes int64) map[string]any {
	return map[string]any{
		"method":         request.Method,
		"path":           request.URL.Path,
		"url":            request.URL.RequestURI(),
		"headers":        redactHeaders(request.Header),
		"body":           sampleString(body, maxBodyBytes),
		"body_truncated": int64(len(body)) > maxBodyBytes,
	}
}

func buildResponseSample(response *http.Response, body []byte, maxBodyBytes int64, envelope map[string]any, hasEnvelope bool) map[string]any {
	sample := map[string]any{
		"status_code":    response.StatusCode,
		"headers":        redactHeaders(response.Header),
		"body":           sampleString(body, maxBodyBytes),
		"body_truncated": int64(len(body)) > maxBodyBytes,
	}
	addEnvelopeFields(sample, envelope, hasEnvelope)
	return sample
}

func addEnvelopeFields(target map[string]any, envelope map[string]any, ok bool) {
	target["nucleus_envelope"] = ok
	if !ok {
		return
	}
	if code, exists := envelope["code"]; exists {
		target["envelope_code"] = code
	}
	if message, exists := envelope["message"]; exists {
		target["envelope_message"] = message
	}
	if traceID, exists := envelope["trace_id"]; exists {
		target["trace_id"] = traceID
	}
	if data, exists := envelope["data"]; exists {
		target["envelope_data"] = data
	}
}

func unwrapEnvelope(body []byte) (map[string]any, bool) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, false
	}
	codeRaw, ok := raw["code"]
	if !ok {
		return nil, false
	}
	var code int
	if err := json.Unmarshal(codeRaw, &code); err != nil {
		return nil, false
	}
	envelope := map[string]any{"code": code}
	if messageRaw, ok := raw["message"]; ok {
		var message string
		if err := json.Unmarshal(messageRaw, &message); err == nil {
			envelope["message"] = message
		}
	}
	if traceRaw, ok := raw["trace_id"]; ok {
		var traceID string
		if err := json.Unmarshal(traceRaw, &traceID); err == nil && traceID != "" {
			envelope["trace_id"] = traceID
		}
	}
	if dataRaw, ok := raw["data"]; ok {
		var data any
		if err := json.Unmarshal(dataRaw, &data); err == nil {
			envelope["data"] = data
		}
	}
	return envelope, true
}

func httpScenarioPass(statusCode int, envelope map[string]any, hasEnvelope bool) (bool, string) {
	if statusCode < 200 || statusCode >= 300 {
		return false, "HTTP status is not 2xx"
	}
	if !hasEnvelope {
		return false, "response is not a Nucleus JSON envelope"
	}
	if code, _ := envelope["code"].(int); code != 0 {
		return false, "Nucleus envelope code is not 0"
	}
	return true, ""
}

func httpAssertionActual(statusCode int, envelope map[string]any, hasEnvelope bool) string {
	if !hasEnvelope {
		return fmt.Sprintf("http_status=%d envelope=missing", statusCode)
	}
	return fmt.Sprintf("http_status=%d envelope_code=%v", statusCode, envelope["code"])
}

func assertionResults(assertions []map[string]any) []map[string]any {
	if assertions == nil {
		return []map[string]any{}
	}
	return assertions
}

func failedHTTPBuildStep(id string, route openapi.Route, err error) map[string]any {
	return map[string]any{
		"id":           id,
		"kind":         "http_scenario",
		"method":       route.Method,
		"path":         route.Path,
		"operation_id": route.OperationID,
		"status":       "failed",
		"pass":         false,
		"reason":       err.Error(),
	}
}

func failedHTTPSample(id string, route openapi.Route, err error) map[string]any {
	return map[string]any{
		"id":           id,
		"method":       route.Method,
		"path":         route.Path,
		"operation_id": route.OperationID,
		"request":      map[string]any{},
		"response":     map[string]any{"error": err.Error()},
	}
}

func requestHookSample(id string, route openapi.Route, request *http.Request, body []byte, err error, maxBodyBytes int64) map[string]any {
	return map[string]any{
		"id":           id,
		"method":       route.Method,
		"path":         route.Path,
		"operation_id": route.OperationID,
		"request":      buildRequestSample(request, body, maxBodyBytes),
		"response":     map[string]any{"error": err.Error()},
	}
}

func failedHTTPAssertion(id string, reason string) map[string]any {
	return map[string]any{
		"id":       id + ".nucleus_envelope",
		"type":     "http_nucleus_envelope",
		"expected": "2xx HTTP status and Nucleus envelope code == 0",
		"actual":   reason,
		"pass":     false,
	}
}

func readBody(body io.Reader, maxBodyBytes int64) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	return io.ReadAll(io.LimitReader(body, maxBodyBytes+1))
}

func sampleString(body []byte, maxBodyBytes int64) string {
	if int64(len(body)) <= maxBodyBytes {
		return string(body)
	}
	return string(body[:maxBodyBytes])
}

func redactHeaders(headers http.Header) map[string][]string {
	redacted := make(map[string][]string, len(headers))
	for key, values := range headers {
		if isSensitiveHeader(key) {
			redacted[key] = []string{"<redacted>"}
			continue
		}
		copied := append([]string(nil), values...)
		redacted[key] = copied
	}
	return redacted
}

func isSensitiveHeader(key string) bool {
	switch strings.ToLower(key) {
	case "authorization", "cookie", "set-cookie":
		return true
	default:
		return false
	}
}

func durationMilliseconds(duration time.Duration) float64 {
	return float64(duration.Microseconds()) / 1000
}
