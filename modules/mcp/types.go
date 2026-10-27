// Package mcp implements the Model Context Protocol (MCP) protocol engine
// for Tracks applications: JSON-RPC 2.0 message handling, a tool registry
// with first-class destructive-tool filtering, a stdio transport, and a
// stateless Streamable HTTP transport that can be mounted on any route.
//
// The package is domain-agnostic: hosts register their own tools via
// RegisterTool and mount the endpoint with Register (or ServeHTTP-style
// helpers). See module.go for the Tracks router integration.
package mcp

import (
	"context"
	"encoding/json"
)

// JSONRPCRequest represents an incoming JSON-RPC 2.0 message
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents an outgoing JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any           `json:"id"`
	Result  any           `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents a standard JSON-RPC 2.0 error object
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Standard JSON-RPC error codes
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// InitializeParams contains client info sent during MCP handshake
type InitializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ClientInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
}

// InitializeResult contains server capabilities and metadata
type InitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ServerInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

// Tool represents a registered MCP tool available for invocation. A tool
// flagged Destructive is dropped from a read-only server at registration
// time, so it can never be listed or invoked through that server.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Destructive bool        `json:"-"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema defines the JSON schema for tool arguments
type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

// Property represents a single parameter definition in an input schema
type Property struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

// ToolHandler defines the function signature for executing an MCP tool
type ToolHandler func(ctx context.Context, args json.RawMessage) (*ToolCallResult, error)

// ToolCallParams represents params passed to tools/call
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// ToolCallResult represents the content payload returned to client
type ToolCallResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ContentItem represents a formatted text/image item in tool results
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ToolsListResult is the result for tools/list
type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}
