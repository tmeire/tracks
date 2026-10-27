package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tmeire/tracks"
)

// mountTestServer wires a read-only server with a fake tool into a fresh
// Tracks test app and returns the routed handler.
func mountTestServer(t *testing.T) (http.Handler, *Server) {
	t.Helper()
	app := tracks.NewTestApp(t, tracks.TestConfig{})
	router := app.Router()

	s := New(t.Context(), WithReadOnly())
	s.RegisterTool(Tool{
		Name:        "ping_tool",
		Description: "Returns pong",
		InputSchema: InputSchema{Type: "object"},
	}, func(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
		return JSONResult(map[string]any{"pong": true}), nil
	})

	Register(router, s, "/test/mcp", ServerInfo{
		Name:     "test-mcp",
		Version:  "0.0.1",
		Endpoint: "/test/mcp",
	}, NoAuth)

	h, err := router.Handler()
	if err != nil {
		t.Fatalf("router handler: %v", err)
	}
	return h, s
}

func post(t *testing.T, h http.Handler, body string, ct string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/test/mcp", strings.NewReader(body))
	if ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestHTTPDiscovery(t *testing.T) {
	h, _ := mountTestServer(t)

	req := httptest.NewRequest("GET", "/test/mcp", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("discovery payload is not JSON: %v (%s)", err, rr.Body.String())
	}
	if payload["read_only"] != true {
		t.Error("expected read_only=true")
	}
	if payload["transport"] != "streamable-http" {
		t.Errorf("expected streamable-http transport, got %v", payload["transport"])
	}
}

func TestHTTPTwoActionsOnSamePath(t *testing.T) {
	// GET and POST must both route to the same path without conflict.
	h, _ := mountTestServer(t)

	rr := post(t, h, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, "application/json")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on POST tools/list, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp JSONRPCResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("not a JSON-RPC response: %v (%s)", err, rr.Body.String())
	}
	if resp.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %v", resp.Error.Message)
	}
}

func TestHTTPToolCall(t *testing.T) {
	h, _ := mountTestServer(t)

	rr := post(t, h, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"ping_tool","arguments":{}}}`, "application/json")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "pong") {
		t.Errorf("expected pong in response, got %s", rr.Body.String())
	}
}

func TestHTTPWrongContentType(t *testing.T) {
	h, _ := mountTestServer(t)

	rr := post(t, h, `{"jsonrpc":"2.0","id":1,"method":"ping"}`, "application/x-www-form-urlencoded")
	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", rr.Code)
	}
}

func TestHTTPSSERefused(t *testing.T) {
	h, _ := mountTestServer(t)

	req := httptest.NewRequest("GET", "/test/mcp", nil)
	req.Header.Set("Accept", "text/event-stream")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for SSE negotiation, got %d", rr.Code)
	}
	if allow := rr.Header().Get("Allow"); allow != "POST" {
		t.Errorf("expected Allow: POST header, got %q", allow)
	}
}

func TestHTTPNotificationNoContent(t *testing.T) {
	h, _ := mountTestServer(t)

	rr := post(t, h, `{"jsonrpc":"2.0","method":"notifications/initialized"}`, "application/json")
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for notification, got %d", rr.Code)
	}
}

func TestRegisterRejectsOpenEndpoint(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected Register to panic without auth middleware")
		}
	}()

	app := tracks.NewTestApp(t, tracks.TestConfig{})
	s := New(t.Context())
	Register(app.Router(), s, "/open/mcp", ServerInfo{Endpoint: "/open/mcp"})
}
