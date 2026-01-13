package mcp

import (
	"encoding/json"
)

// JSON-RPC 2.0 message types

// Request represents a JSON-RPC request
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC response
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error represents a JSON-RPC error
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// MCP-specific message types

// InitializeResult is the response to the initialize request
type InitializeResult struct {
	ProtocolVersion string               `json:"protocolVersion"`
	ServerInfo      ServerInfo           `json:"serverInfo"`
	Capabilities    ServerCapabilities   `json:"capabilities"`
}

// ServerInfo contains information about the server
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerCapabilities describes what the server can do
type ServerCapabilities struct {
	Tools ToolsCapability `json:"tools"`
}

// ToolsCapability describes tool capabilities
type ToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

// Tool represents an MCP tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolCallParams represents parameters for calling a tool
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult is the response from calling a tool
type ToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content represents content in a tool result
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// NewResponse creates a JSON-RPC response
func NewResponse(id interface{}, result interface{}) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

// NewErrorResponse creates a JSON-RPC error response
func NewErrorResponse(id interface{}, code int, message string) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
}

// NewToolResult creates a tool result with text content
func NewToolResult(text string) *ToolResult {
	return &ToolResult{
		Content: []Content{
			{
				Type: "text",
				Text: text,
			},
		},
	}
}

// NewErrorToolResult creates an error tool result
func NewErrorToolResult(errorMsg string) *ToolResult {
	return &ToolResult{
		Content: []Content{
			{
				Type: "text",
				Text: errorMsg,
			},
		},
		IsError: true,
	}
}
