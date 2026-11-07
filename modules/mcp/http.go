package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/tmeire/tracks"
)

// ServerInfo describes an MCP endpoint for discovery responses.
type ServerInfo struct {
	// Name and Version identify the MCP server implementation.
	Name    string
	Version string
	// Endpoint is the HTTP path the JSON-RPC endpoint is mounted on.
	Endpoint string
	// Issuer is the public origin of this server (scheme + host, no trailing
	// slash). When empty it is derived from the router's config.
	Issuer string
	// OAuthBase is the public path prefix for the OAuth endpoints (e.g.
	// "/mcp" when the MCP resource lives under an authenticated area such as
	// /admin). The authorize/token/register endpoints are mounted at
	// OAuthBase + "/oauth/...". Leave empty to disable OAuth.
	OAuthBase string
}

// issuerFromRouter resolves the public origin, preferring an explicit
// ServerInfo.Issuer. The config's port is only appended for plain-HTTP
// (development) servers: behind a TLS terminator the public origin never
// carries the internal port, so proxied hosts must set ServerInfo.Issuer.
func issuerFromRouter(r tracks.Router, info ServerInfo) string {
	if info.Issuer != "" {
		return strings.TrimRight(info.Issuer, "/")
	}
	scheme := "http"
	if r.Secure() {
		scheme = "https"
	}
	host := r.BaseDomain()
	if port := r.Config().Port; !r.Secure() && port != 0 && port != 80 {
		host = fmt.Sprintf("%s:%d", host, port)
	}
	return scheme + "://" + host
}

// IsSSERequest reports whether the request negotiates a server-sent-events
// stream (used by stateful Streamable HTTP clients on GET).
func IsSSERequest(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "text/event-stream")
}

// DiscoveryPayload builds the JSON discovery document describing this
// server: its identity, transport posture, and tool list.
func (s *Server) DiscoveryPayload(info ServerInfo) map[string]any {
	tools := s.Tools()
	return map[string]any{
		"server":        info.Name,
		"version":       info.Version,
		"protocol":      LatestProtocolVersion,
		"transport":     "streamable-http",
		"endpoint":      info.Endpoint,
		"read_only":     s.readOnly,
		"oauth":         s.OAuthEnabled(),
		"tools":         tools,
		"tools_enabled": len(tools),
	}
}

// jsonMediaType normalizes a Content-Type header to its bare media type.
func jsonMediaType(ct string) string {
	return strings.TrimSpace(strings.SplitN(ct, ";", 2)[0])
}

// HTTPHandler returns a plain net/http handler implementing the stateless
// subset of the Streamable HTTP transport:
//
//   - POST with a JSON-RPC 2.0 body is answered with a single JSON response
//     (or 204 No Content for notifications).
//   - GET with Accept: text/event-stream is refused with 405 (stateless).
//   - GET otherwise returns the discovery document.
//
// Hosts are responsible for authentication; wrap this handler in whatever
// middleware applies (e.g. session admin guards).
func (s *Server) HTTPHandler(info ServerInfo) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if IsSSERequest(r) {
				w.Header().Set("Allow", "POST")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusMethodNotAllowed)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "This MCP endpoint is stateless and does not support SSE streams; POST JSON-RPC requests instead.",
				})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(s.DiscoveryPayload(info))

		case http.MethodPost:
			if ct := jsonMediaType(r.Header.Get("Content-Type")); ct != "application/json" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnsupportedMediaType)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "Content-Type must be application/json"})
				return
			}

			body, err := io.ReadAll(io.LimitReader(r.Body, 10*1024*1024))
			if err != nil || len(body) == 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "Empty or unreadable request body"})
				return
			}

			resp, err := s.HandleJSONRPC(r.Context(), body)
			if err != nil {
				if err == ErrNoResponse {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				slog.Error("Failed to process MCP JSON-RPC request", "error", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "An internal server error occurred"})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)

		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// Register mounts the stateless Streamable HTTP MCP endpoint on the Tracks
// router at path (e.g. "/admin/mcp"). The auth builders, if any, are applied
// to the discovery, consent and authorize actions; the endpoint refuses to
// mount without at least one, so an unauthenticated MCP endpoint cannot be
// registered by accident. For a deliberately open endpoint (e.g. local
// development), pass NoAuth explicitly.
//
// When the server was created with WithOAuth, info.OAuthBase selects the
// public path prefix for the OAuth endpoints (use a path outside the host's
// authenticated area, e.g. "/mcp" for "/admin/mcp"): consent runs in the
// admin's browser behind the auth builders, while token exchange and client
// registration are public per the OAuth spec. The JSON-RPC POST then accepts
// a valid Bearer access token OR the host's session cookie; without either
// it answers 401 with a WWW-Authenticate header pointing at the protected
// resource metadata, which is what MCP clients use to start the OAuth flow.
func Register(r tracks.Router, server *Server, path string, info ServerInfo, auth ...tracks.MiddlewareBuilder) tracks.Router {
	if len(auth) == 0 {
		panic("mcp: refusing to mount an MCP endpoint without authentication middleware; pass NoAuth to override")
	}

	issuer := issuerFromRouter(r, info)
	slog.Info("Registering MCP endpoint (Streamable HTTP, stateless)",
		"path", path, "read_only", server.readOnly, "oauth", server.OAuthEnabled())

	// Refuse SSE stream negotiation on GET with a proper 405. This runs as
	// middleware because Tracks' content negotiation would otherwise answer
	// 406 before the action handler runs.
	sseGuard := func(tracks.Router) tracks.Middleware {
		return func(next http.Handler) (http.Handler, error) {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodGet && IsSSERequest(req) {
					w.Header().Set("Allow", "POST")
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusMethodNotAllowed)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"error": "This MCP endpoint is stateless and does not support SSE streams; POST JSON-RPC requests instead.",
					})
					return
				}
				next.ServeHTTP(w, req)
			}), nil
		}
	}

	// GET path: discovery document (authenticated).
	r.Serve(tracks.Action{
		Method:      http.MethodGet,
		Path:        path,
		Controller:  "mcp",
		Name:        "discovery",
		Func:        discoveryFunc(server, info),
		Layout:      "plain",
		Middlewares: append([]tracks.MiddlewareBuilder{sseGuard}, auth...),
	})

	// POST path: JSON-RPC 2.0. With OAuth, a valid access token replaces the
	// session; without any credentials the session middleware decides, and a
	// login redirect is converted to the spec's 401 + WWW-Authenticate so
	// clients can start the OAuth flow.
	postAuth := auth
	if server.OAuthEnabled() {
		postAuth = []tracks.MiddlewareBuilder{server.BearerOrSession(info, auth)}
	}
	r.Serve(tracks.Action{
		Method:      http.MethodPost,
		Path:        path,
		Controller:  "mcp",
		Name:        "rpc",
		Func:        rpcFunc(server),
		Layout:      "plain",
		Middlewares: append([]tracks.MiddlewareBuilder{sseGuard}, postAuth...),
	})

	if !server.OAuthEnabled() || info.OAuthBase == "" {
		return r
	}

	oauthBase := "/" + strings.Trim(info.OAuthBase, "/")
	mountOAuth(r, server, path, info, issuer, oauthBase, auth)
	return r
}

// mountOAuth wires the authorization server endpoints and the well-known
// metadata documents. The authorize endpoints run behind the host's auth
// builders (consent is only given by an authenticated admin); token exchange
// and dynamic client registration are public per the OAuth spec.
func mountOAuth(r tracks.Router, server *Server, path string, info ServerInfo, issuer, oauthBase string, auth []tracks.MiddlewareBuilder) {
	r.Serve(tracks.Action{
		Method:      http.MethodGet,
		Path:        oauthBase + "/oauth/authorize",
		Controller:  "mcp",
		Name:        "oauth_consent",
		Func:        server.oauthAuthorizeGet(issuer, oauthBase),
		Layout:      "plain",
		Middlewares: auth,
	})
	r.Serve(tracks.Action{
		Method:      http.MethodPost,
		Path:        oauthBase + "/oauth/authorize",
		Controller:  "mcp",
		Name:        "oauth_consent",
		Func:        server.oauthAuthorizePost(issuer),
		Layout:      "plain",
		Middlewares: auth,
	})
	r.Serve(tracks.Action{
		Method:     http.MethodPost,
		Path:       oauthBase + "/oauth/token",
		Controller: "mcp",
		Name:       "oauth_token",
		Func:       server.oauthTokenHandler(issuer),
		Layout:     "plain",
	})
	r.Serve(tracks.Action{
		Method:     http.MethodPost,
		Path:       oauthBase + "/oauth/register",
		Controller: "mcp",
		Name:       "oauth_register",
		Func:       server.oauthRegisterHandler(),
		Layout:     "plain",
	})
	r.Serve(tracks.Action{
		Method:     http.MethodGet,
		Path:       "/.well-known/oauth-authorization-server",
		Controller: "mcp",
		Name:       "oauth_server_metadata",
		Func:       metadataFunc(server.authorizationServerMetadata(issuer, oauthBase)),
		Layout:     "plain",
	})
	r.Serve(tracks.Action{
		Method:     http.MethodGet,
		Path:       "/.well-known/oauth-authorization-server" + oauthBase,
		Controller: "mcp",
		Name:       "oauth_server_metadata",
		Func:       metadataFunc(server.authorizationServerMetadata(issuer, oauthBase)),
		Layout:     "plain",
	})
	r.Serve(tracks.Action{
		Method:     http.MethodGet,
		Path:       "/.well-known/oauth-protected-resource" + path,
		Controller: "mcp",
		Name:       "oauth_resource_metadata",
		Func:       metadataFunc(server.protectedResourceMetadata(issuer, path, oauthBase)),
		Layout:     "plain",
	})
}

// metadataFunc serves a static JSON document.
func metadataFunc(doc map[string]any) tracks.ActionFunc {
	return func(r *http.Request) (any, error) {
		return &tracks.Response{StatusCode: http.StatusOK, Data: doc}, nil
	}
}

// NoAuth explicitly marks an MCP endpoint as unauthenticated. Only use this
// for local development or when an outer layer performs authentication.
func NoAuth(tracks.Router) tracks.Middleware {
	return func(next http.Handler) (http.Handler, error) {
		return next, nil
	}
}

func discoveryFunc(server *Server, info ServerInfo) tracks.ActionFunc {
	return func(r *http.Request) (any, error) {
		return &tracks.Response{
			StatusCode: http.StatusOK,
			Data:       server.DiscoveryPayload(info),
		}, nil
	}
}

func rpcFunc(server *Server) tracks.ActionFunc {
	return func(r *http.Request) (any, error) {
		if ct := jsonMediaType(r.Header.Get("Content-Type")); ct != "application/json" {
			return &tracks.Response{
				StatusCode: http.StatusUnsupportedMediaType,
				Data:       map[string]string{"error": "Content-Type must be application/json"},
			}, nil
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, 10*1024*1024))
		if err != nil || len(body) == 0 {
			return &tracks.Response{
				StatusCode: http.StatusBadRequest,
				Data:       map[string]string{"error": "Empty or unreadable request body"},
			}, nil
		}

		// The request context carries the host application's database where
		// Tracks injects it, so tools run against it directly.
		resp, err := server.HandleJSONRPC(r.Context(), body)
		if err != nil {
			if err == ErrNoResponse {
				return &tracks.Response{StatusCode: http.StatusNoContent}, nil
			}
			slog.Error("Failed to process MCP JSON-RPC request", "error", err)
			return &tracks.Response{
				StatusCode: http.StatusInternalServerError,
				Data:       map[string]string{"error": "An internal server error occurred"},
			}, nil
		}

		return &tracks.Response{
			StatusCode: http.StatusOK,
			Data:       resp,
		}, nil
	}
}
