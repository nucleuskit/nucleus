package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Server exposes local Nucleus facts as MCP tools.
type Server struct {
	dir string
}

// NewServer creates an MCP server rooted at a service directory.
func NewServer(dir string) *Server {
	if strings.TrimSpace(dir) == "" {
		dir = defaultDir
	}
	return &Server{dir: dir}
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type callToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type toolCallResult struct {
	Content           []toolContent `json:"content"`
	StructuredContent any           `json:"structuredContent,omitempty"`
	IsError           bool          `json:"isError,omitempty"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Serve reads MCP JSON-RPC messages using Content-Length framing.
func (server *Server) Serve(input io.Reader, output io.Writer) error {
	reader := bufio.NewReader(input)
	for {
		payload, err := readFrame(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		response, ok := server.Handle(payload)
		if !ok {
			continue
		}
		if err := writeFrame(output, response); err != nil {
			return err
		}
	}
}

// Handle processes one JSON-RPC request payload. It returns ok=false for notifications.
func (server *Server) Handle(payload []byte) ([]byte, bool) {
	var request rpcRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		return mustJSON(rpcResponse{JSONRPC: jsonRPCVersion, Error: &rpcError{Code: -32700, Message: "parse error"}}), true
	}
	if request.ID == nil {
		_, _ = server.dispatch(request)
		return nil, false
	}
	result, rpcErr := server.dispatch(request)
	response := rpcResponse{JSONRPC: jsonRPCVersion, ID: request.ID, Result: result, Error: rpcErr}
	if rpcErr != nil {
		response.Result = nil
	}
	return mustJSON(response), true
}

func (server *Server) dispatch(request rpcRequest) (any, *rpcError) {
	switch request.Method {
	case "initialize":
		return map[string]any{
			"protocolVersion": nucleusMCPProtocol,
			"serverInfo": map[string]any{
				"name":    serverName,
				"version": serverVersion,
			},
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
		}, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return map[string]any{"tools": server.Tools()}, nil
	case "tools/call":
		var params callToolParams
		if len(request.Params) > 0 {
			if err := json.Unmarshal(request.Params, &params); err != nil {
				return nil, &rpcError{Code: -32602, Message: "invalid tools/call params"}
			}
		}
		payload, err := server.CallTool(params.Name, params.Arguments)
		if err != nil {
			return toolResult(map[string]any{"error": err.Error()}, true), nil
		}
		return toolResult(payload, false), nil
	case "notifications/initialized":
		return map[string]any{}, nil
	default:
		return nil, &rpcError{Code: -32601, Message: "method not found"}
	}
}

func toolResult(payload any, isError bool) toolCallResult {
	data, err := json.Marshal(payload)
	if err != nil {
		data = []byte(`{"error":"tool payload could not be encoded"}`)
		isError = true
	}
	return toolCallResult{
		Content:           []toolContent{{Type: "text", Text: string(data)}},
		StructuredContent: payload,
		IsError:           isError,
	}
}

func readFrame(reader *bufio.Reader) ([]byte, error) {
	contentLength := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			parsed, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, fmt.Errorf("invalid content length: %w", err)
			}
			contentLength = parsed
		}
	}
	if contentLength <= 0 {
		return nil, fmt.Errorf("missing content length")
	}
	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func writeFrame(output io.Writer, payload []byte) error {
	_, err := fmt.Fprintf(output, "Content-Length: %d\r\n\r\n%s", len(payload), payload)
	return err
}

func mustJSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		return []byte(`{"jsonrpc":"2.0","error":{"code":-32603,"message":"internal error"}}`)
	}
	return data
}

func frame(payload []byte) []byte {
	var buffer bytes.Buffer
	_, _ = fmt.Fprintf(&buffer, "Content-Length: %d\r\n\r\n", len(payload))
	buffer.Write(payload)
	return buffer.Bytes()
}
