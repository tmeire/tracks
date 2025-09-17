package tracks

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tmeire/tracks/session"
	"github.com/tmeire/tracks/session/inmemory"
)

func TestViewVars_NilContext(t *testing.T) {
	if got := ViewVars(nil); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}

func TestAddViewVar_AddsAndOverwrites(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	r2 := AddViewVar(r, "foo", 1)
	if r2 == r {
		t.Fatalf("expected a new request with updated context")
	}

	// Old request should not have the var
	if m := ViewVars(r.Context()); m != nil && m["foo"] != nil {
		t.Fatalf("original request unexpectedly has view var")
	}

	// New request should have the var
	if got := ViewVars(r2.Context())["foo"]; got != 1 {
		t.Fatalf("expected foo=1 on updated request, got %#v", got)
	}

	// Overwrite
	r3 := AddViewVar(r2, "foo", 2)
	if got := ViewVars(r3.Context())["foo"]; got != 2 {
		t.Fatalf("expected foo=2 after overwrite, got %#v", got)
	}
}

func TestActionRenderHTML_ExposesVarsAndVFunc(t *testing.T) {
	// Build a minimal template with a "page" definition using .Vars and v
	tplText := `{{ define "page" }}A={{ .Vars.a }}, B={{ v "b" }}{{ end }}`
	tpl := templateMustParse(tplText)

	a := &action{template: tpl}

	// Prepare request with view vars
	baseReq := httptest.NewRequest(http.MethodGet, "/", nil)
	baseReq = AddViewVar(baseReq, "a", "alpha")
	baseReq = AddViewVar(baseReq, "b", "bravo")

	// Wrap a handler calling renderHTML with session middleware so Flash/Session access is safe
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

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, baseReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "A=alpha") || !strings.Contains(body, "B=bravo") {
		t.Fatalf("unexpected body: %q", body)
	}
}

// templateMustParse is a tiny helper to build a template with a "page" definition
// that renderHTML expects. We don't add the "t" func as the test does not call it.
func templateMustParse(text string) *template.Template {
	// Register a placeholder for v so the parser accepts templates using it.
	t := template.New("test").Funcs(template.FuncMap{
		"v": func(string, ...any) any { return nil },
	})
	tpl, err := t.Parse(text)
	if err != nil {
		panic(err)
	}
	return tpl
}
