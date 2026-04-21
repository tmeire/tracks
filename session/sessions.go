package session

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Store interface {
	Load(ctx context.Context, id string) (Session, bool)
	Create(ctx context.Context) Session
}

// Session represents a user sessions with methods for managing sessions data.
type Session interface {
	Authenticate(userId string)
	Authenticated() (string, bool)
	IsAuthenticated() bool
	// Get retrieves a value from the sessions by key.
	// If the key doesn't exist, it returns nil.
	Get(key string) (value string, ok bool)

	// Put stores a value in the sessions by key.
	Put(key string, value string)

	// Forget removes a key from the sessions.
	Forget(key string)

	// ID returns the sessions ID.
	ID() string

	// Flash adds a flash message to the sessions.
	Flash(key string, value string)

	// FlashMessages returns all flash messages from the previous request.
	FlashMessages() map[string]string

	// Save persists the current sessions state to the underlying store.
	Save(ctx context.Context) error
	Invalidate(ctx context.Context)
}

// Middleware is a middleware that adds a sessions to the request context.
func Middleware(domain string, store Store) func(next http.Handler) (http.Handler, error) {
	if host, _, err := net.SplitHostPort(domain); err == nil {
		domain = host
	}

	cookieDomain := domain
	if domain != "localhost" && !strings.Contains(domain, ":") && !net.ParseIP(domain).To4().Equal(net.IPv4(127, 0, 0, 1)) {
		// Use leading dot for domain cookies to ensure they are sent to subdomains
		// but only if it's not localhost or an IP
		if strings.Contains(domain, ".") {
			cookieDomain = "." + domain
		}
	}

	return func(next http.Handler) (http.Handler, error) {
		return &middleware{
			domain: cookieDomain,
			store:  store,
			next:   next,
		}, nil
	}
}

// Context key for the sessions
type contextKey string

const sessionKey contextKey = "sessions"

// FromRequest retrieves the sessions from the request context.
func FromRequest(r *http.Request) Session {
	return FromContext(r.Context())
}

// FromContext retrieves the sessions from the context.
func FromContext(r context.Context) Session {
	s := r.Value(sessionKey)
	if s == nil {
		return nil
	}

	if session, ok := s.(Session); ok {
		return session
	}
	return nil
}

func Flash(r *http.Request, key, value string) {
	session := FromRequest(r)
	session.Flash(key, value)
}

func Invalidate(r *http.Request) {
	session := FromRequest(r)
	session.Invalidate(r.Context())
}

func FlashMessages(r *http.Request) map[string]string {
	session := FromRequest(r)
	return session.FlashMessages()
}

type middleware struct {
	domain string
	store  Store
	next   http.Handler
}

type sessionResponseWriter struct {
	http.ResponseWriter
	session    Session
	initialID  string
	domain     string
	ctx        context.Context
	secure     bool
	saved      bool
}

func (w *sessionResponseWriter) WriteHeader(code int) {
	if !w.saved {
		w.save()
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *sessionResponseWriter) Write(b []byte) (int, error) {
	if !w.saved {
		w.save()
	}
	return w.ResponseWriter.Write(b)
}

func (w *sessionResponseWriter) save() {
	if w.session.ID() != w.initialID {
		// Session ID changed (e.g. invalidated), set a new cookie
		http.SetCookie(w.ResponseWriter, &http.Cookie{
			Name:     "sessions",
			Value:    w.session.ID(),
			Domain:   w.domain,
			Path:     "/",
			HttpOnly: true,
			Secure:   w.secure,
			SameSite: http.SameSiteLaxMode,
		})
	}

	err := w.session.Save(w.ctx)
	if err != nil {
		slog.ErrorContext(w.ctx, "Failed to save session", "session_id", w.session.ID(), "error", err)
		span := trace.SpanFromContext(w.ctx)
		span.RecordError(err)
	}
	w.saved = true
}

func (m *middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if FromRequest(r) != nil {
		m.next.ServeHTTP(w, r)
		return
	}

	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(r.Context(), "session")
	defer span.End()

	r = r.WithContext(ctx)

	session := m.load(span, w, r)

	ctx = context.WithValue(ctx, sessionKey, session)
	r = r.WithContext(ctx)

	session.Put("ip", r.RemoteAddr)
	session.Put("user_agent", r.UserAgent())

	sw := &sessionResponseWriter{
		ResponseWriter: w,
		session:        session,
		initialID:      session.ID(),
		domain:         m.domain,
		ctx:            ctx,
		secure:         IsSecure(r),
	}

	m.next.ServeHTTP(sw, r)

	if !sw.saved {
		sw.save()
	}
}

func (m *middleware) load(span trace.Span, w http.ResponseWriter, r *http.Request) Session {
	var sess Session
	cookie, err := r.Cookie("sessions")
	if err == nil {
		span.SetAttributes(attribute.String("sessions.id", cookie.Value))
		session, ok := m.store.Load(r.Context(), cookie.Value)
		if ok {
			sess = session
		} else {
			slog.WarnContext(r.Context(), "Session cookie present but session not found in store", "session_id", cookie.Value)
			span.AddEvent("unknown-session")
		}
	} else {
		span.AddEvent("no-session")
	}

	if sess == nil {
		sess = m.store.Create(r.Context())
		span.SetAttributes(attribute.String("sessions.id", sess.ID()))
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "sessions",
		Value:    sess.ID(),
		Domain:   m.domain,
		Path:     "/",
		HttpOnly: true,
		Secure:   IsSecure(r),
		SameSite: http.SameSiteLaxMode,
	})

	return sess
}

// IsSecure returns true if the request is served over HTTPS or if it's behind a proxy that terminates TLS.
func IsSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		return true
	}
	return false
}
