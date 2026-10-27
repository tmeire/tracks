package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

func testServer(t *testing.T, opts ...Option) (*Server, context.Context) {
	t.Helper()
	s := New(context.Background(), opts...)
	s.RegisterTool(Tool{
		Name:        "echo",
		Description: "Returns its arguments",
		InputSchema: InputSchema{Type: "object"},
	}, func(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
		return JSONResult(map[string]any{"echoed": string(args)}), nil
	})
	s.RegisterTool(Tool{
		Name:        "delete_everything",
		Description: "Irreversible test tool",
		Destructive: true,
		InputSchema: InputSchema{Type: "object"},
	}, func(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
		return JSONResult(map[string]any{"deleted": true}), nil
	})
	return s, context.Background()
}

func rpc(t *testing.T, s *Server, ctx context.Context, req map[string]any) JSONRPCResponse {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	resp, err := s.HandleJSONRPC(ctx, body)
	if err != nil {
		t.Fatalf("HandleJSONRPC: %v", err)
	}
	return resp
}

func TestDestructiveToolsDroppedOnReadOnlyServer(t *testing.T) {
	full, _ := testServer(t)
	ro, _ := testServer(t, WithReadOnly())

	if len(full.Tools()) != 2 {
		t.Fatalf("expected 2 tools on full server, got %d", len(full.Tools()))
	}
	tools := ro.Tools()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool on read-only server, got %d", len(tools))
	}
	if tools[0].Name != "echo" {
		t.Fatalf("expected only 'echo' to survive, got %q", tools[0].Name)
	}
	if !ro.ReadOnly() {
		t.Error("expected ReadOnly() to be true")
	}

	// A direct call to the destructive tool must fail with method-not-found
	resp := rpc(t, ro, context.Background(), map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params":  map[string]any{"name": "delete_everything", "arguments": map[string]any{}},
	})
	if resp.Error == nil || resp.Error.Code != CodeMethodNotFound {
		t.Fatalf("expected method-not-found for destructive tool, got %+v", resp)
	}
}

func TestInitializeProtocolVersionNegotiation(t *testing.T) {
	s, _ := testServer(t)

	cases := []struct {
		requested string
		expected  string
	}{
		{"2025-06-18", "2025-06-18"},
		{"2025-03-26", "2025-03-26"},
		{"2024-11-05", "2024-11-05"},
		{"1999-01-01", LatestProtocolVersion},
		{"", LatestProtocolVersion},
	}

	for _, tc := range cases {
		resp := rpc(t, s, context.Background(), map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params":  map[string]any{"protocolVersion": tc.requested},
		})
		resultBytes, _ := json.Marshal(resp.Result)
		var result InitializeResult
		if err := json.Unmarshal(resultBytes, &result); err != nil {
			t.Fatalf("unmarshal initialize result: %v", err)
		}
		if result.ProtocolVersion != tc.expected {
			t.Errorf("requested %q: got %q, want %q", tc.requested, result.ProtocolVersion, tc.expected)
		}
	}
}

func TestHandleJSONRPCParseError(t *testing.T) {
	s, _ := testServer(t)
	resp, err := s.HandleJSONRPC(context.Background(), []byte("not json"))
	if err != nil {
		t.Fatalf("HandleJSONRPC: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != CodeParseError {
		t.Fatalf("expected parse error, got %+v", resp)
	}
}

func TestHandleJSONRPCNotification(t *testing.T) {
	s, _ := testServer(t)
	resp, err := s.HandleJSONRPC(context.Background(), []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	if err != ErrNoResponse {
		t.Fatalf("expected ErrNoResponse, got %v", err)
	}
	if resp.ID != nil || resp.Result != nil {
		t.Fatalf("expected empty response, got %+v", resp)
	}
}

func TestToolCallRoundTrip(t *testing.T) {
	s, _ := testServer(t)
	resp := rpc(t, s, context.Background(), map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params":  map[string]any{"name": "echo", "arguments": map[string]any{"x": 1}},
	})
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error.Message)
	}
	resultBytes, _ := json.Marshal(resp.Result)
	var result ToolCallResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(result.Content) != 1 || result.Content[0].Type != "text" {
		t.Fatalf("unexpected result content: %+v", result)
	}
	var echoed map[string]any
	if err := json.Unmarshal([]byte(result.Content[0].Text), &echoed); err != nil {
		t.Fatalf("tool result is not the echoed JSON: %s", result.Content[0].Text)
	}
	if _, ok := echoed["echoed"]; !ok {
		t.Fatalf("expected echoed payload, got %s", result.Content[0].Text)
	}
}

func TestStdioServe(t *testing.T) {
	s, _ := testServer(t)

	in := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n")
	out := &bytes.Buffer{}
	if err := s.Serve(in, out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response %q: %v", out.String(), err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error.Message)
	}
}
