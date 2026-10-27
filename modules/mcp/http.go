package mcp

import (
	"encoding/json"
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
// to both the GET and POST actions; the endpoint refuses to mount without at
// least one, so an unauthenticated MCP endpoint cannot be registered by
// accident. For a deliberately open endpoint (e.g. local development), pass
// NoAuth explicitly.
func Register(r tracks.Router, server *Server, path string, info ServerInfo, auth ...tracks.MiddlewareBuilder) tracks.Router {
	if len(auth) == 0 {
		panic("mcp: refusing to mount an MCP endpoint without authentication middleware; pass NoAuth to override")
	}

	slog.Info("Registering MCP endpoint (Streamable HTTP, stateless)", "path", path, "read_only", server.ReadOnly())

	// Refuse SSE stream negotiation on GET with a proper 405. This runs as
	// middleware because Tracks' content negotiation would otherwise answer
	// 406 before the action handler runs.
	auth = append(auth, func(tracks.Router) tracks.Middleware {
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
	})

	// GET path: discovery document.
	r.Serve(tracks.Action{
		Method:      http.MethodGet,
		Path:        path,
		Controller:  "mcp",
		Name:        "discovery",
		Func:        discoveryFunc(server, info),
		Layout:      "plain",
		Middlewares: auth,
	})

	// POST path: JSON-RPC 2.0.
	r.Serve(tracks.Action{
		Method:      http.MethodPost,
		Path:        path,
		Controller:  "mcp",
		Name:        "rpc",
		Func:        rpcFunc(server),
		Layout:      "plain",
		Middlewares: auth,
	})

	return r
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
