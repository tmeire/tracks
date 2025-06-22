package session

import (
	"context"
	"net"
	"net/http"

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
	domain, _, _ = net.SplitHostPort(domain)

	return func(next http.Handler) (http.Handler, error) {
		return &middleware{
			domain: "." + domain,
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

func (m *middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.GetTracerProvider().Tracer("tracks").Start(r.Context(), "session")
	defer span.End()

	r = r.WithContext(ctx)

	session := m.load(span, w, r)

	ctx = context.WithValue(ctx, sessionKey, session)
	r = r.WithContext(ctx)

	m.next.ServeHTTP(w, r)

	err := session.Save(ctx)
	if err != nil {
		panic(err)
	}
}

func (m *middleware) load(span trace.Span, w http.ResponseWriter, r *http.Request) Session {
	cookie, err := r.Cookie("sessions")
	if err == nil {
		span.SetAttributes(attribute.String("sessions.id", cookie.Value))
		session, ok := m.store.Load(r.Context(), cookie.Value)
		if ok {
			return session
		}
		span.AddEvent("unknown-session")
	}
	span.AddEvent("no-session")

	session := m.store.Create(r.Context())
	span.SetAttributes(attribute.String("sessions.id", session.ID()))

	http.SetCookie(w, &http.Cookie{
		Name:     "sessions",
		Value:    session.ID(),
		Domain:   m.domain,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	return session
}
