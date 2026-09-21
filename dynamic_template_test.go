package tracks

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tmeire/tracks/session"
	"github.com/tmeire/tracks/session/inmemory"
)

func TestActionRenderHTML_DynamicTemplateMultipleRequests(t *testing.T) {
	// Construct a minimal template with a "page" definition
	tplText := `{{ define "page" }}Hello, Dynamic!{{ end }}`
	tOriginal := template.New("test").Funcs(template.FuncMap{
		"v": func(string, ...any) any { return nil },
		"t": func(string, ...any) any { return nil },
		"safe": func(string) any { return nil },
		"csrf_token": func() string { return "" },
		"csrf_field": func() any { return nil },
	})
	defaultTpl, err := tOriginal.Parse(tplText)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}

	dt := &dynamicTemplate{
		defaultTpl: defaultTpl,
	}

	a := &action{
		template: dt,
	}

	// Session middleware setup
	store := inmemory.NewStore()
	mw := session.Middleware("localhost:8080", store)

	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := &Response{StatusCode: http.StatusOK}
		if err := a.renderHTML(r, w, resp); err != nil {
			t.Fatalf("renderHTML error: %v", err)
		}
	})

	handler, err := mw(h)
	if err != nil {
		t.Fatalf("failed to build session middleware: %v", err)
	}

	// First Request: Should succeed
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("First request: expected status 200, got %d", w1.Code)
	}

	// Second Request: Previously this would panic or fail due to tpl.Funcs call on the same template pointer.
	// With the clone fix, it must succeed without panic.
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	w2 := httptest.NewRecorder()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Second request panicked: %v", r)
		}
	}()

	handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("Second request: expected status 200, got %d", w2.Code)
	}
}
