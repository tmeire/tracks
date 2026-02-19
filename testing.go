package tracks

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tmeire/tracks/database"
	"github.com/tmeire/tracks/session"
	sessionconfig "github.com/tmeire/tracks/session/config"
)

type TestApp struct {
	t          *testing.T
	router     Router
	testUserID string
}

type TestConfig struct {
	Database      string
	Transactional bool
}

func NewTestApp(t *testing.T, config TestConfig) *TestApp {
	conf := Config{
		Port: 8080,
		Database: database.Config{
			Type:   "sqlite",
			Config: []byte(`{"path": ":memory:"}`),
		},
		Sessions: sessionconfig.Config{
			Store: struct {
				Type   string          `json:"type"`
				Config json.RawMessage `json:"config"`
			}{
				Type: "inmemory",
			},
		},
	}

	app := &TestApp{
		t:      t,
		router: NewFromConfig(context.Background(), conf),
	}

	// Add test middleware for authentication injection
	app.router.GlobalMiddleware(func(next http.Handler) (http.Handler, error) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if app.testUserID != "" {
				sess := session.FromRequest(r)
				if sess != nil {
					sess.Authenticate(app.testUserID)
				}
			}
			next.ServeHTTP(w, r)
		}), nil
	})

	return app
}

func (a *TestApp) Router() Router {
	return a.router
}

func (a *TestApp) DB() database.Database {
	return a.router.Database()
}

func (a *TestApp) AuthenticateAs(userID string) {
	a.testUserID = userID
}

func (a *TestApp) PerformRequest(method, path string, body io.Reader, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	h, err := a.router.Handler()
	if err != nil {
		a.t.Fatalf("Failed to get handler: %v", err)
	}

	h.ServeHTTP(w, req)
	return w
}

func (a *TestApp) Get(path string) *httptest.ResponseRecorder {
	return a.PerformRequest("GET", path, nil, nil)
}

type JSONBody map[string]any

func (a *TestApp) PostJSON(path string, body JSONBody) *httptest.ResponseRecorder {
	b, err := json.Marshal(body)
	if err != nil {
		a.t.Fatalf("Failed to marshal JSON: %v", err)
	}

	return a.PerformRequest("POST", path, bytes.NewBuffer(b), map[string]string{
		"Content-Type": "application/json",
	})
}

func (a *TestApp) Post(path string, body io.Reader) *httptest.ResponseRecorder {
	return a.PerformRequest("POST", path, body, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	})
}
