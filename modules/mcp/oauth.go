package mcp

// OAuth 2.1 authorization server for the MCP endpoint, so MCP clients can
// connect through the browser consent flow instead of a shared session
// cookie. Implements what the MCP auth spec (2025-06-18) expects of an
// authorization server protecting a resource:
//
//   - RFC 7591 dynamic client registration (public clients only)
//   - RFC 7636 authorization code + PKCE (S256 required)
//   - RFC 6749 token endpoint (authorization_code and refresh_token grants)
//   - RFC 8414 authorization server metadata and RFC 9728 protected
//     resource metadata under /.well-known/
//
// Grants, tokens and registered clients are kept in memory: they survive as
// long as the process does, which is the right trade-off for an admin
// endpoint (a restart logs clients out and they silently re-consent).
// Consent is expressed by an authenticated admin in their browser; the host
// mounts the authorize endpoints behind its own auth middleware (see
// Register) and provides the consent view.

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/tmeire/tracks"
)

const (
	accessTokenTTL  = time.Hour
	refreshTokenTTL = 30 * 24 * time.Hour
	authCodeTTL     = 10 * time.Minute
	pendingAuthzTTL = 10 * time.Minute
	// mcpScope is the only scope this server grants; it is echoed but not enforced
	// because the tool set is already restricted to read-only.
	mcpScope = "mcp"
)

// oauthClient is a dynamically registered public client.
type oauthClient struct {
	ID           string
	Name         string
	RedirectURIs []string
	CreatedAt    time.Time
}

// pendingAuthz is an authorization request awaiting consent. The nonce binds
// the approval POST to the consent page the admin actually saw, and
// ConsentSubject lets the host record which admin approved (token audit).
type pendingAuthz struct {
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	Nonce               string
	ConsentSubject      func(r *http.Request) string
	ExpiresAt           time.Time
}

// authCode is a single-use authorization code.
type authCode struct {
	ClientID      string
	RedirectURI   string
	Scope         string
	CodeChallenge string
	Subject       string
	ExpiresAt     time.Time
}

// oauthToken is an issued access or refresh token, keyed by SHA-256 hash.
type oauthToken struct {
	Hash      string
	Type      string // "access" or "refresh"
	Subject   string
	ClientID  string
	Scope     string
	ExpiresAt time.Time
}

// OAuthStore holds OAuth clients, pending consent requests and tokens in memory.
type OAuthStore struct {
	mu sync.Mutex
	// ConsentSubject optionally extracts the authenticated admin's ID from the
	// consent request for the token audit trail; see WithOAuth.
	ConsentSubject func(r *http.Request) string
	now            func() time.Time
	clients        map[string]*oauthClient
	pending        map[string]*pendingAuthz
	codes          map[string]*authCode
	tokens         map[string]*oauthToken
}

func newOAuthStore() *OAuthStore {
	return &OAuthStore{
		now:     time.Now,
		clients: map[string]*oauthClient{},
		pending: map[string]*pendingAuthz{},
		codes:   map[string]*authCode{},
		tokens:  map[string]*oauthToken{},
	}
}

func randomToken(prefix string) string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b)
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (s *OAuthStore) sweep(now time.Time) {
	for id, p := range s.pending {
		if now.After(p.ExpiresAt) {
			delete(s.pending, id)
		}
	}
	for id, c := range s.codes {
		if now.After(c.ExpiresAt) {
			delete(s.codes, id)
		}
	}
	for h, t := range s.tokens {
		if now.After(t.ExpiresAt) {
			delete(s.tokens, h)
		}
	}
}

// RegisterClient stores a dynamically registered public client.
func (s *OAuthStore) RegisterClient(name string, redirectURIs []string) *oauthClient {
	s.mu.Lock()
	defer s.mu.Unlock()
	client := &oauthClient{
		ID:           randomToken("mcp-client-"),
		Name:         oauthTruncate(name, 128),
		RedirectURIs: redirectURIs,
		CreatedAt:    s.now(),
	}
	s.clients[client.ID] = client
	return client
}

// Client returns a registered client by ID.
func (s *OAuthStore) Client(id string) *oauthClient {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.clients[id]
}

// SavePending stores a pending authorization request and returns its ID.
func (s *OAuthStore) SavePending(p *pendingAuthz) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweep(s.now())
	id := randomToken("mcp-req-")
	p.ExpiresAt = s.now().Add(pendingAuthzTTL)
	s.pending[id] = p
	return id
}

// Pending returns a still-valid pending authorization request.
func (s *OAuthStore) Pending(id string) *pendingAuthz {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.pending[id]
	if !ok || s.now().After(p.ExpiresAt) {
		return nil
	}
	return p
}

// TakePending removes and returns a pending authorization request, so its
// nonce and request ID can each be used exactly once.
func (s *OAuthStore) TakePending(id string) *pendingAuthz {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.pending[id]
	if !ok || s.now().After(p.ExpiresAt) {
		return nil
	}
	delete(s.pending, id)
	return p
}

// IssueCode creates a single-use authorization code for an approved request.
func (s *OAuthStore) IssueCode(p *pendingAuthz, subject string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	code := randomToken("mcp-code-")
	s.sweep(s.now())
	s.codes[code] = &authCode{
		ClientID:      p.ClientID,
		RedirectURI:   p.RedirectURI,
		Scope:         p.Scope,
		CodeChallenge: p.CodeChallenge,
		Subject:       subject,
		ExpiresAt:     s.now().Add(authCodeTTL),
	}
	return code
}

// TakeCode consumes a single-use authorization code.
func (s *OAuthStore) TakeCode(code string) *authCode {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.codes[code]
	if !ok {
		return nil
	}
	delete(s.codes, code)
	if s.now().After(c.ExpiresAt) {
		return nil
	}
	return c
}

// issueTokenPair mints an access/refresh token pair. Caller must hold the lock.
func (s *OAuthStore) issueTokenPairLocked(c *authCode) (access, refresh string) {
	s.sweep(s.now())
	access = randomToken("mcp-at-")
	s.tokens[tokenHash(access)] = &oauthToken{
		Hash: tokenHash(access), Type: "access", Subject: c.Subject,
		ClientID: c.ClientID, Scope: c.Scope, ExpiresAt: s.now().Add(accessTokenTTL),
	}
	refresh = randomToken("mcp-rt-")
	s.tokens[tokenHash(refresh)] = &oauthToken{
		Hash: tokenHash(refresh), Type: "refresh", Subject: c.Subject,
		ClientID: c.ClientID, Scope: c.Scope, ExpiresAt: s.now().Add(refreshTokenTTL),
	}
	return access, refresh
}

// ExchangeCode redeems an authorization code for a token pair.
func (s *OAuthStore) ExchangeCode(c *authCode) (access, refresh string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.issueTokenPairLocked(c)
}

// Refresh validates a refresh token and rotates it into a new pair.
func (s *OAuthStore) Refresh(refreshToken string) (access, refresh string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, exists := s.tokens[tokenHash(refreshToken)]
	if !exists || t.Type != "refresh" || s.now().After(t.ExpiresAt) {
		return "", "", false
	}
	// Rotation: the used refresh token can never be replayed.
	delete(s.tokens, tokenHash(refreshToken))
	c := &authCode{ClientID: t.ClientID, Subject: t.Subject, Scope: t.Scope}
	a, r := s.issueTokenPairLocked(c)
	return a, r, true
}

// ValidateAccess returns the subject of a valid, unexpired access token.
func (s *OAuthStore) ValidateAccess(accessToken string) (subject string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, exists := s.tokens[tokenHash(accessToken)]
	if !exists || t.Type != "access" || s.now().After(t.ExpiresAt) {
		return "", false
	}
	return t.Subject, true
}

// validRedirectURI enforces https anywhere and http on loopback only (RFC 8252),
// plus private-use custom schemes (com.example.app:/) that the OS routes.
func validRedirectURI(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Fragment != "" || u.Scheme == "" {
		return false
	}
	switch u.Scheme {
	case "https":
		return true
	case "http":
		host := u.Hostname()
		return host == "localhost" || host == "127.0.0.1" || host == "::1"
	default:
		// Private-use schemes (RFC 8252 §7.3): reverse-domain notation.
		return strings.Contains(u.Scheme, ".")
	}
}

// oauthTruncate limits a stored string length.
func oauthTruncate(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

// --- Endpoint actions ------------------------------------------------------

// oauthAuthorizeGet validates the authorization request, stores it as pending
// and renders the consent page. The host view (mcp/oauth_consent) shows the
// client name and posts the decision back.
func (s *Server) oauthAuthorizeGet(issuer, oauthBase string) tracks.ActionFunc {
	return func(r *http.Request) (any, error) {
		q := r.URL.Query()
		if q.Get("response_type") != "code" {
			return oauthError("", issuer, "unsupported_response_type", "")
		}
		client := s.oauth.Client(q.Get("client_id"))
		if client == nil {
			return &tracks.Response{StatusCode: http.StatusBadRequest, Data: map[string]string{
				"error": "unknown client_id",
			}}, nil
		}
		redirectURI := q.Get("redirect_uri")
		if !slicesContains(client.RedirectURIs, redirectURI) {
			// Never redirect to an unregistered URI; tell the admin in place.
			return &tracks.Response{StatusCode: http.StatusBadRequest, Data: map[string]string{
				"error": "redirect_uri is not registered for this client",
			}}, nil
		}
		if q.Get("code_challenge") == "" || q.Get("code_challenge_method") != "S256" {
			return oauthError(redirectURI, issuer, "invalid_request", "code_challenge_method must be S256")
		}

		pending := &pendingAuthz{
			ClientID:            client.ID,
			RedirectURI:         redirectURI,
			Scope:               q.Get("scope"),
			State:               q.Get("state"),
			CodeChallenge:       q.Get("code_challenge"),
			CodeChallengeMethod: q.Get("code_challenge_method"),
			Nonce:               randomToken("mcp-nonce-"),
			ConsentSubject:      s.oauth.ConsentSubject,
		}
		requestID := s.oauth.SavePending(pending)

		r = tracks.AddViewVar(r, "ClientName", client.Name)
		r = tracks.AddViewVar(r, "Scopes", pending.Scope)
		r = tracks.AddViewVar(r, "RequestID", requestID)
		r = tracks.AddViewVar(r, "Nonce", pending.Nonce)
		r = tracks.AddViewVar(r, "ActionURL", issuer+oauthBase+"/oauth/authorize")
		return nil, nil
	}
}

// oauthAuthorizePost consumes the consent decision and redirects back to the
// client with a code (or an access_denied error).
func (s *Server) oauthAuthorizePost(issuer string) tracks.ActionFunc {
	return func(r *http.Request) (any, error) {
		if err := r.ParseForm(); err != nil {
			return &tracks.Response{StatusCode: http.StatusBadRequest, Data: map[string]string{"error": "invalid form"}}, nil
		}
		p := s.oauth.TakePending(r.Form.Get("request_id"))
		if p == nil || subtle.ConstantTimeCompare([]byte(r.Form.Get("nonce")), []byte(p.Nonce)) != 1 {
			return &tracks.Response{StatusCode: http.StatusBadRequest, Data: map[string]string{
				"error": "Unknown or expired authorization request. Start the connection again.",
			}}, nil
		}

		if r.Form.Get("decision") != "approve" {
			return oauthError(p.RedirectURI, issuer, "access_denied", p.State)
		}

		// The consent page is only reachable by an authenticated admin (the
		// route runs behind the host's auth middleware); the subject is the
		// admin the token will act as, recorded for auditing.
		subject := "admin"
		if p.ConsentSubject != nil {
			if id := p.ConsentSubject(r); id != "" {
				subject = id
			}
		}

		code := s.oauth.IssueCode(p, subject)
		return oauthCodeRedirect(p, issuer, code), nil
	}
}

// oauthCodeRedirect sends the browser back to the client with the code.
func oauthCodeRedirect(p *pendingAuthz, issuer, code string) *tracks.Response {
	u, _ := url.Parse(p.RedirectURI)
	q := u.Query()
	q.Set("code", code)
	if p.State != "" {
		q.Set("state", p.State)
	}
	q.Set("iss", issuer)
	u.RawQuery = q.Encode()
	return &tracks.Response{StatusCode: http.StatusFound, Location: u.String()}
}

// oauthError builds an error redirect to the client callback, or a plain JSON
// error when no redirect target is trustworthy.
func oauthError(redirectURI, issuer, code, state string) (*tracks.Response, error) {
	if redirectURI == "" {
		return &tracks.Response{StatusCode: http.StatusBadRequest, Data: map[string]string{"error": code}}, nil
	}
	u, _ := url.Parse(redirectURI)
	q := u.Query()
	q.Set("error", code)
	if state != "" {
		q.Set("state", state)
	}
	q.Set("iss", issuer)
	u.RawQuery = q.Encode()
	return &tracks.Response{StatusCode: http.StatusFound, Location: u.String()}, nil
}

func slicesContains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

// oauthTokenHandler implements the token endpoint.
func (s *Server) oauthTokenHandler(issuer string) tracks.ActionFunc {
	return func(r *http.Request) (any, error) {
		if err := r.ParseForm(); err != nil {
			return tokenError("invalid_request")
		}

		switch r.Form.Get("grant_type") {
		case "authorization_code":
			code := s.oauth.TakeCode(r.Form.Get("code"))
			if code == nil {
				return tokenError("invalid_grant")
			}
			client := s.oauth.Client(r.Form.Get("client_id"))
			if client == nil || client.ID != code.ClientID {
				return tokenError("invalid_client")
			}
			if code.RedirectURI != r.Form.Get("redirect_uri") {
				return tokenError("invalid_grant")
			}
			// PKCE: S256(verifier) must match the challenge from the authorize request.
			verifier := r.Form.Get("code_verifier")
			if len(verifier) < 43 || len(verifier) > 128 {
				return tokenError("invalid_grant")
			}
			sum := sha256.Sum256([]byte(verifier))
			if subtle.ConstantTimeCompare(
				[]byte(base64.RawURLEncoding.EncodeToString(sum[:])),
				[]byte(code.CodeChallenge)) != 1 {
				return tokenError("invalid_grant")
			}
			access, refresh := s.oauth.ExchangeCode(code)
			return tokenResponse(access, refresh, code.Scope), nil

		case "refresh_token":
			access, refresh, ok := s.oauth.Refresh(r.Form.Get("refresh_token"))
			if !ok {
				return tokenError("invalid_grant")
			}
			return tokenResponse(access, refresh, ""), nil

		default:
			return tokenError("unsupported_grant_type")
		}
	}
}

func tokenError(code string) (*tracks.Response, error) {
	return &tracks.Response{StatusCode: http.StatusBadRequest, Data: map[string]string{"error": code}}, nil
}

type tokenResponsePayload struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

func tokenResponse(access, refresh, scope string) tokenResponsePayload {
	return tokenResponsePayload{
		AccessToken:  access,
		TokenType:    "Bearer",
		ExpiresIn:    int(accessTokenTTL.Seconds()),
		RefreshToken: refresh,
		Scope:        scope,
	}
}

// oauthRegisterHandler implements RFC 7591 dynamic client registration for
// public clients.
func (s *Server) oauthRegisterHandler() tracks.ActionFunc {
	return func(r *http.Request) (any, error) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
		if err != nil {
			return &tracks.Response{StatusCode: http.StatusBadRequest, Data: map[string]string{"error": "invalid_client_metadata"}}, nil
		}
		var meta struct {
			ClientName              string   `json:"client_name"`
			RedirectURIs            []string `json:"redirect_uris"`
			TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
			GrantTypes              []string `json:"grant_types"`
			ResponseTypes           []string `json:"response_types"`
		}
		if err := json.Unmarshal(body, &meta); err != nil || len(meta.RedirectURIs) == 0 {
			return &tracks.Response{StatusCode: http.StatusBadRequest, Data: map[string]string{"error": "invalid_redirect_uri"}}, nil
		}
		for _, uri := range meta.RedirectURIs {
			if !validRedirectURI(uri) {
				return &tracks.Response{StatusCode: http.StatusBadRequest, Data: map[string]string{"error": "invalid_redirect_uri"}}, nil
			}
		}
		if meta.TokenEndpointAuthMethod != "" && meta.TokenEndpointAuthMethod != "none" {
			return &tracks.Response{StatusCode: http.StatusBadRequest, Data: map[string]string{"error": "invalid_client_metadata"}}, nil
		}

		client := s.oauth.RegisterClient(meta.ClientName, meta.RedirectURIs)
		return &tracks.Response{StatusCode: http.StatusCreated, Data: map[string]any{
			"client_id":                  client.ID,
			"client_name":                client.Name,
			"client_id_issued_at":        client.CreatedAt.Unix(),
			"redirect_uris":              client.RedirectURIs,
			"token_endpoint_auth_method": "none",
			"grant_types":                []string{"authorization_code", "refresh_token"},
			"response_types":             []string{"code"},
			"scope":                      mcpScope,
		}}, nil
	}
}

// --- Metadata documents ----------------------------------------------------

// authorizationServerMetadata is the RFC 8414 document for this server.
func (s *Server) authorizationServerMetadata(issuer, oauthBase string) map[string]any {
	return map[string]any{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + oauthBase + "/oauth/authorize",
		"token_endpoint":                        issuer + oauthBase + "/oauth/token",
		"registration_endpoint":                 issuer + oauthBase + "/oauth/register",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"scopes_supported":                      []string{mcpScope},
	}
}

// protectedResourceMetadata is the RFC 9728 document for the MCP resource.
func (s *Server) protectedResourceMetadata(issuer, endpointPath, oauthBase string) map[string]any {
	return map[string]any{
		"resource":                 issuer + endpointPath,
		"authorization_servers":    []string{issuer},
		"scopes_supported":         []string{mcpScope},
		"bearer_methods_supported": []string{"header"},
	}
}

// --- Auth ------------------------------------------------------------------

// BearerOrSession returns middleware for the JSON-RPC POST route that accepts
// either a valid OAuth access token or the host's own session (admin cookie).
// A session redirect (visitor not logged in) is converted to the spec's
// 401 + WWW-Authenticate so MCP clients can start the OAuth flow; other
// session outcomes (e.g. 403 for a non-admin) pass through unchanged.
func (s *Server) BearerOrSession(info ServerInfo, sessionAuth []tracks.MiddlewareBuilder) tracks.MiddlewareBuilder {
	return func(router tracks.Router) tracks.Middleware {
		wwwAuth := fmt.Sprintf(`Bearer resource_metadata="%s/.well-known/oauth-protected-resource%s"`,
			issuerFromRouter(router, info), info.Endpoint)
		sessionMws := make([]tracks.Middleware, len(sessionAuth))
		for i, b := range sessionAuth {
			sessionMws[i] = b(router)
		}
		return func(next http.Handler) (http.Handler, error) {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if token := BearerToken(r); token != "" {
					if _, ok := s.oauth.ValidateAccess(token); !ok {
						unauthorized(w, wwwAuth)
						return
					}
					next.ServeHTTP(w, r)
					return
				}

				// No token: fall back to the host session middleware.
				iw := &interceptRedirect{ResponseWriter: w, wwwAuth: wwwAuth}
				h := http.Handler(next)
				for i := len(sessionMws) - 1; i >= 0; i-- {
					wrapped, err := sessionMws[i](h)
					if err != nil {
						slog.Error("Failed to build MCP session middleware", "error", err)
						http.Error(w, "An internal server error occurred", http.StatusInternalServerError)
						return
					}
					h = wrapped
				}
				h.ServeHTTP(iw, r)
			}), nil
		}
	}
}

func unauthorized(w http.ResponseWriter, wwwAuth string) {
	w.Header().Set("WWW-Authenticate", wwwAuth)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized", "error_description": "Provide a valid Bearer access token or an authenticated admin session."})
}

// interceptRedirect swallows a redirect response and replaces it with a
// 401 + WWW-Authenticate, so the session middleware's login redirect becomes
// the signal MCP clients use to start the OAuth flow.
type interceptRedirect struct {
	http.ResponseWriter
	wwwAuth     string
	intercepted bool
}

func (w *interceptRedirect) WriteHeader(status int) {
	if status >= 300 && status < 400 && !w.intercepted {
		w.intercepted = true
		w.Header().Del("Location")
		unauthorized(w.ResponseWriter, w.wwwAuth)
		return
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *interceptRedirect) Write(b []byte) (int, error) {
	if w.intercepted {
		return len(b), nil // the redirected body is replaced by the 401 JSON
	}
	return w.ResponseWriter.Write(b)
}
func BearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

// ValidateBearer returns the token subject when the request carries a valid
// access token.
func (s *Server) ValidateBearer(r *http.Request) (string, bool) {
	if s.oauth == nil {
		return "", false
	}
	token := BearerToken(r)
	if token == "" {
		return "", false
	}
	subject, ok := s.oauth.ValidateAccess(token)
	return subject, ok
}
