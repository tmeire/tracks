package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/tmeire/tracks"
)

// mountOAuthTestApp wires a read-only OAuth-enabled server into a fresh test
// app. The session auth is NoAuth, so the consent flow is open in tests; the
// browser-side cookie behavior is covered by the app's own tests.
func mountOAuthTestApp(t *testing.T) (http.Handler, *Server) {
	t.Helper()
	app := tracks.NewTestApp(t, tracks.TestConfig{})
	router := app.Router()

	s := New(t.Context(), WithReadOnly(), WithOAuth(nil))
	s.RegisterTool(Tool{
		Name:        "ping_tool",
		Description: "Returns pong",
		InputSchema: InputSchema{Type: "object"},
	}, func(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
		return JSONResult(map[string]any{"pong": true}), nil
	})

	Register(router, s, "/admin/mcp", ServerInfo{
		Name:      "test-mcp",
		Version:   "0.0.1",
		Endpoint:  "/admin/mcp",
		Issuer:    "https://mcp.example.com",
		OAuthBase: "/mcp",
	}, NoAuth)

	h, err := router.Handler()
	if err != nil {
		t.Fatalf("router handler: %v", err)
	}
	return h, s
}

func mustOK(t *testing.T, rr *httptest.ResponseRecorder, where string) {
	t.Helper()
	if rr.Code != http.StatusOK {
		t.Fatalf("%s: expected 200, got %d: %s", where, rr.Code, rr.Body.String())
	}
}

func do(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestOAuthFullFlow(t *testing.T) {
	h, _ := mountOAuthTestApp(t)

	// 0. Metadata documents are discoverable.
	rr := do(h, mustRequest("GET", "/.well-known/oauth-protected-resource/admin/mcp", nil))
	mustOK(t, rr, "protected resource metadata")
	var resourceMeta struct {
		Resource             string   `json:"resource"`
		AuthorizationServers []string `json:"authorization_servers"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resourceMeta); err != nil {
		t.Fatalf("resource metadata not JSON: %v (%s)", err, rr.Body.String())
	}
	if resourceMeta.Resource != "https://mcp.example.com/admin/mcp" || len(resourceMeta.AuthorizationServers) == 0 {
		t.Fatalf("unexpected resource metadata: %+v", resourceMeta)
	}

	rr = do(h, mustRequest("GET", "/.well-known/oauth-authorization-server/mcp", nil))
	mustOK(t, rr, "authorization server metadata")
	var asMeta struct {
		Issuer               string   `json:"issuer"`
		RegistrationEndpoint string   `json:"registration_endpoint"`
		TokenEndpoint        string   `json:"token_endpoint"`
		CodeChallengeMethods []string `json:"code_challenge_methods_supported"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &asMeta); err != nil {
		t.Fatalf("as metadata not JSON: %v (%s)", err, rr.Body.String())
	}
	if asMeta.Issuer == "" || asMeta.TokenEndpoint == "" {
		t.Fatalf("unexpected AS metadata: %+v", asMeta)
	}

	// 1. Dynamic client registration.
	rr = do(h, mustRequest("POST", "/mcp/oauth/register", strings.NewReader(
		`{"client_name":"Claude","redirect_uris":["https://claude.ai/api/mcp/auth/callback"],"token_endpoint_auth_method":"none"}`)))
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 on registration, got %d: %s", rr.Code, rr.Body.String())
	}
	var client struct {
		ClientID string `json:"client_id"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &client); err != nil || client.ClientID == "" {
		t.Fatalf("no client_id in registration response: %s", rr.Body.String())
	}

	// 2. Authorization request renders the consent page with a nonce.
	verifier := base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("v", 64)))
	challenge := base64.RawURLEncoding.EncodeToString(func() []byte {
		sum := sha256.Sum256([]byte(verifier))
		return sum[:]
	}())
	authzURL := "/mcp/oauth/authorize?" + url.Values{
		"response_type":         {"code"},
		"client_id":             {client.ClientID},
		"redirect_uri":          {"https://claude.ai/api/mcp/auth/callback"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"state":                 {"st-123"},
		"scope":                 {"mcp"},
	}.Encode()
	rr = do(h, mustRequest("GET", authzURL, nil))
	mustOK(t, rr, "authorize (consent page)")
	requestID := extractInput(t, rr.Body.String(), "request_id")
	nonce := extractInput(t, rr.Body.String(), "nonce")
	if requestID == "" || nonce == "" {
		t.Fatalf("consent page missing request_id/nonce: %s", rr.Body.String())
	}

	// 3. Approval redirects to the client with a code (and state).
	rr = do(h, mustRequest("POST", "/mcp/oauth/authorize",
		strings.NewReader(url.Values{
			"request_id": {requestID},
			"nonce":      {nonce},
			"decision":   {"approve"},
		}.Encode())))
	if rr.Code != http.StatusFound {
		t.Fatalf("expected 302 after approval, got %d: %s", rr.Code, rr.Body.String())
	}
	callback, err := url.Parse(rr.Header().Get("Location"))
	if err != nil {
		t.Fatalf("invalid redirect: %v", err)
	}
	if callback.Query().Get("state") != "st-123" {
		t.Errorf("state not echoed: %s", rr.Header().Get("Location"))
	}
	code := callback.Query().Get("code")
	if code == "" {
		t.Fatalf("no code in redirect: %s", rr.Header().Get("Location"))
	}

	// 4. Token exchange with the PKCE verifier.
	rr = do(h, mustRequest("POST", "/mcp/oauth/token",
		strings.NewReader(url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"client_id":     {client.ClientID},
			"redirect_uri":  {"https://claude.ai/api/mcp/auth/callback"},
			"code_verifier": {verifier},
		}.Encode())))
	mustOK(t, rr, "token exchange")
	var token struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &token); err != nil || token.AccessToken == "" {
		t.Fatalf("no access token: %s", rr.Body.String())
	}

	// The code is single-use: redeeming it again fails.
	rr = do(h, mustRequest("POST", "/mcp/oauth/token",
		strings.NewReader(url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"client_id":     {client.ClientID},
			"redirect_uri":  {"https://claude.ai/api/mcp/auth/callback"},
			"code_verifier": {verifier},
		}.Encode())))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on code replay, got %d: %s", rr.Code, rr.Body.String())
	}

	// 5. A wrong verifier is rejected.
	rr = do(h, mustRequest("GET", authzURL, nil))
	requestID2 := extractInput(t, rr.Body.String(), "request_id")
	nonce2 := extractInput(t, rr.Body.String(), "nonce")
	rr = do(h, mustRequest("POST", "/mcp/oauth/authorize",
		strings.NewReader(url.Values{"request_id": {requestID2}, "nonce": {nonce2}, "decision": {"approve"}}.Encode())))
	code2 := callbackQuery(t, rr.Header().Get("Location"), "code")
	rr = do(h, mustRequest("POST", "/mcp/oauth/token",
		strings.NewReader(url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code2},
			"client_id":     {client.ClientID},
			"redirect_uri":  {"https://claude.ai/api/mcp/auth/callback"},
			"code_verifier": {base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("x", 64)))},
		}.Encode())))
	if rr.Code != http.StatusBadRequest || !strings.Contains(rr.Body.String(), "invalid_grant") {
		t.Fatalf("expected invalid_grant on wrong verifier, got %d: %s", rr.Code, rr.Body.String())
	}

	// 6. Valid token: JSON-RPC works. Invalid token: 401 with WWW-Authenticate.
	rpc := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	req := mustRequest("POST", "/admin/mcp", strings.NewReader(rpc))
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	mustOK(t, rr, "rpc with bearer token")
	if !strings.Contains(rr.Body.String(), "ping_tool") {
		t.Errorf("expected tools in response, got %s", rr.Body.String())
	}

	req = mustRequest("POST", "/admin/mcp", strings.NewReader(rpc))
	req.Header.Set("Authorization", "Bearer mcp-at-invalid")
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid token, got %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("WWW-Authenticate"), "resource_metadata") {
		t.Errorf("expected WWW-Authenticate with resource_metadata, got %q", rr.Header().Get("WWW-Authenticate"))
	}

	// 7. Refresh rotation: old refresh token dies, new one works.
	rr = do(h, mustRequest("POST", "/mcp/oauth/token",
		strings.NewReader(url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {token.RefreshToken},
		}.Encode())))
	mustOK(t, rr, "refresh")
	var refreshed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &refreshed); err != nil || refreshed.AccessToken == "" {
		t.Fatalf("no refreshed token: %s", rr.Body.String())
	}
	rr = do(h, mustRequest("POST", "/mcp/oauth/token",
		strings.NewReader(url.Values{"grant_type": {"refresh_token"}, "refresh_token": {token.RefreshToken}}.Encode())))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on replayed refresh token, got %d", rr.Code)
	}
}

func mustRequest(method, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)
	req.Header.Set("User-Agent", "test-agent")
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	return req
}

func extractInput(t *testing.T, html, name string) string {
	t.Helper()
	marker := `name="` + name + `" value="`
	i := strings.Index(html, marker)
	if i < 0 {
		return ""
	}
	rest := html[i+len(marker):]
	if j := strings.Index(rest, `"`); j >= 0 {
		return rest[:j]
	}
	return ""
}

func callbackQuery(t *testing.T, raw, key string) string {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("bad redirect %q: %v", raw, err)
	}
	return u.Query().Get(key)
}
