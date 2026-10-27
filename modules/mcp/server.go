package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

// ToolHandler is defined in types.go.

// SupportedProtocolVersions lists the MCP protocol versions this server can
// speak, newest last. Clients may request any of these in initialize.
var SupportedProtocolVersions = []string{"2024-11-05", "2025-03-26", "2025-06-18"}

// LatestProtocolVersion is the newest supported MCP protocol version.
const LatestProtocolVersion = "2025-06-18"

// ErrNoResponse signals that a notification was processed and nothing should
// be written back to the client.
var ErrNoResponse = errors.New("mcp: notification processed, no response")

// Option configures a Server at construction time.
type Option func(*Server)

// WithReadOnly marks the server as read-only: tools registered with
// Destructive set are silently dropped, so they can never be listed or
// called through this server. This is the recommended posture for endpoints
// exposed over HTTP.
func WithReadOnly() Option {
	return func(s *Server) {
		s.readOnly = true
	}
}

// Server coordinates JSON-RPC 2.0 MCP communication over any transport. The
// stdio loop (Serve) and the HTTP adapter (see http.go) both build on it.
type Server struct {
	dbCtx    context.Context
	tools    map[string]ToolHandler
	toolDefs []Tool
	readOnly bool
	mu       sync.RWMutex
}

// New instantiates an MCP server. The context is used as the database
// context for the stdio transport; the HTTP transport derives the context
// from each incoming request instead. Register domain tools right after
// construction via RegisterTool.
func New(ctx context.Context, opts ...Option) *Server {
	s := &Server{
		dbCtx:    ctx,
		tools:    make(map[string]ToolHandler),
		toolDefs: []Tool{},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ReadOnly reports whether destructive tools are excluded from this server.
func (s *Server) ReadOnly() bool {
	return s.readOnly
}

// RegisterTool adds a tool definition and handler to the server. On a
// read-only server, destructive tools are silently dropped.
func (s *Server) RegisterTool(def Tool, handler ToolHandler) {
	if s.readOnly && def.Destructive {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.toolDefs = append(s.toolDefs, def)
	s.tools[def.Name] = handler
}

// Tools returns a copy of the registered tool definitions
func (s *Server) Tools() []Tool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	defs := make([]Tool, len(s.toolDefs))
	copy(defs, s.toolDefs)
	return defs
}

// ServeStdio executes the server loop using os.Stdin and os.Stdout.
func (s *Server) ServeStdio() error {
	return s.Serve(os.Stdin, os.Stdout)
}

// Serve processes newline-delimited JSON-RPC requests from in and writes responses to out
func (s *Server) Serve(in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	// Support payloads up to 10MB
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	encoder := json.NewEncoder(out)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			resp := JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      nil,
				Error: &JSONRPCError{
					Code:    CodeParseError,
					Message: "Parse error: invalid JSON",
				},
			}
			_ = encoder.Encode(resp)
			continue
		}

		// Handle notifications (no response required when ID is nil)
		if req.ID == nil {
			s.handleNotification(&req)
			continue
		}

		resp := s.handleRequest(s.dbCtx, &req)
		if err := encoder.Encode(resp); err != nil {
			slog.Error("Failed to write JSON-RPC response", "error", err)
		}
	}

	return scanner.Err()
}

// HandleJSONRPC processes a single JSON-RPC 2.0 request payload and returns
// the response to serialize back to the caller. It is the HTTP counterpart
// of Serve and expects the database to be present in ctx where tools need it.
func (s *Server) HandleJSONRPC(ctx context.Context, body []byte) (JSONRPCResponse, error) {
	var req JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      nil,
			Error: &JSONRPCError{
				Code:    CodeParseError,
				Message: "Parse error: invalid JSON",
			},
		}, nil
	}

	// Notifications get no response per the JSON-RPC spec
	if req.ID == nil {
		s.handleNotification(&req)
		return JSONRPCResponse{}, ErrNoResponse
	}

	return s.handleRequest(ctx, &req), nil
}

func (s *Server) handleNotification(req *JSONRPCRequest) {
	// e.g. "notifications/initialized", "$/cancelRequest"
	switch req.Method {
	case "notifications/initialized":
		slog.Info("MCP client initialized successfully")
	default:
		// Silently ignore unrecognized notifications per protocol
	}
}

func (s *Server) handleRequest(ctx context.Context, req *JSONRPCRequest) JSONRPCResponse {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "initialize":
		var params InitializeParams
		if len(req.Params) > 0 {
			_ = json.Unmarshal(req.Params, &params)
		}
		// Echo back the client's requested version when we support it, so
		// both sides agree; otherwise respond with our latest version and
		// let the client decide whether it can work with it.
		requested := params.ProtocolVersion
		version := LatestProtocolVersion
		for _, v := range SupportedProtocolVersions {
			if requested == v {
				version = requested
				break
			}
		}
		result := InitializeResult{
			ProtocolVersion: version,
			Capabilities: map[string]any{
				"tools": map[string]any{
					"listChanged": false,
				},
			},
		}
		result.ServerInfo.Name = "boldline-mcp"
		result.ServerInfo.Version = "1.0.0"
		resp.Result = result

	case "ping":
		resp.Result = map[string]any{}

	case "tools/list":
		resp.Result = ToolsListResult{
			Tools: s.Tools(),
		}

	case "tools/call":
		var params ToolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			resp.Error = &JSONRPCError{
				Code:    CodeInvalidParams,
				Message: "Invalid params for tools/call",
			}
			return resp
		}

		s.mu.RLock()
		handler, exists := s.tools[params.Name]
		s.mu.RUnlock()

		if !exists {
			resp.Error = &JSONRPCError{
				Code:    CodeMethodNotFound,
				Message: fmt.Sprintf("Tool %q not found", params.Name),
			}
			return resp
		}

		res, err := handler(ctx, params.Arguments)
		if err != nil {
			resp.Result = ToolCallResult{
				Content: []ContentItem{
					{Type: "text", Text: "Error executing tool: " + err.Error()},
				},
				IsError: true,
			}
		} else {
			resp.Result = res
		}

	default:
		resp.Error = &JSONRPCError{
			Code:    CodeMethodNotFound,
			Message: fmt.Sprintf("Method %q not found", req.Method),
		}
	}

	return resp
}

// JSONResult marshals v as the sole text content item of a tool result.
func JSONResult(v any) *ToolCallResult {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ErrorResult("Failed to marshal JSON: " + err.Error())
	}
	return &ToolCallResult{
		Content: []ContentItem{
			{Type: "text", Text: string(bytes)},
		},
	}
}

// ErrorResult builds a tool result flagged as an error.
func ErrorResult(msg string) *ToolCallResult {
	return &ToolCallResult{
		Content: []ContentItem{
			{Type: "text", Text: msg},
		},
		IsError: true,
	}
}
