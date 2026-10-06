package tracks

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tmeire/tracks/i18n"
)

func newLocalizedApp(t *testing.T) *TestApp {
	t.Helper()
	return NewTestApp(t, TestConfig{I18n: &i18n.Config{
		Locales:  []string{"en", "nl"},
		Strategy: i18n.StrategyPath,
		BaseURL:  "https://example.com",
	}})
}

func TestPathLocales_OneRouteServesEveryLanguage(t *testing.T) {
	app := newLocalizedApp(t)
	app.Router().GetFunc("/coloring-pages/{slug}", "catalog", "show", func(r *http.Request) (any, error) {
		return map[string]string{
			"path":      r.URL.Path,
			"slug":      r.PathValue("slug"),
			"lang":      i18n.LanguageFromContext(r.Context()),
			"cart":      LocalizePath(r, "/cart"),
			"canonical": i18n.CanonicalURL(r.Context()),
		}, nil
	})

	tests := []struct {
		url  string
		want map[string]string
	}{
		{"/coloring-pages/cow", map[string]string{"path": "/coloring-pages/cow", "slug": "cow", "lang": "en", "cart": "/cart", "canonical": "https://example.com/coloring-pages/cow"}},
		{"/nl/coloring-pages/koe", map[string]string{"path": "/coloring-pages/koe", "slug": "koe", "lang": "nl", "cart": "/nl/cart", "canonical": "https://example.com/nl/coloring-pages/koe"}},
	}
	for _, tt := range tests {
		w := app.PerformRequest(http.MethodGet, tt.url, nil, map[string]string{"Accept": "application/json"})
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status %d: %s", tt.url, w.Code, w.Body)
		}
		var got map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatalf("%s: %v", tt.url, err)
		}
		for k, v := range tt.want {
			if got[k] != v {
				t.Errorf("%s: %s = %q, want %q", tt.url, k, got[k], v)
			}
		}
	}

	if w := app.Get("/en/coloring-pages/cow"); w.Code != http.StatusMovedPermanently || w.Header().Get("Location") != "/coloring-pages/cow" {
		t.Errorf("/en prefix: got %d %q", w.Code, w.Header().Get("Location"))
	}
}

func TestPathLocales_RedirectsStayInLanguage(t *testing.T) {
	app := newLocalizedApp(t)
	app.Router().
		GetFunc("/checkout", "checkout", "new", Redirect("/cart")).
		GetFunc("/to-english", "locale", "switch", func(r *http.Request) (any, error) {
			return &Response{StatusCode: http.StatusSeeOther, Location: i18n.URLFor(r.Context(), "en", "/cart")}, nil
		}).
		Redirect("/old", "/new")

	tests := []struct{ url, want string }{
		{"/checkout", "/cart"},
		{"/nl/checkout", "/nl/cart"},
		{"/nl/to-english", "https://example.com/cart"},
		{"/old", "/new"},
		{"/nl/old", "/nl/new"},
	}
	for _, tt := range tests {
		w := app.Get(tt.url)
		if loc := w.Header().Get("Location"); loc != tt.want {
			t.Errorf("%s: Location = %q (status %d), want %q", tt.url, loc, w.Code, tt.want)
		}
	}
}

func TestPathLocales_ViewVars(t *testing.T) {
	app := newLocalizedApp(t)
	app.Router().GetFunc("/", "home", "index", func(r *http.Request) (any, error) {
		return map[string]any{
			"locale":    ViewVar(r, "locale"),
			"suggested": ViewVar(r, "suggested_locale"),
			"legacy":    ViewVar(r, "canonical_nl"),
		}, nil
	})

	w := app.PerformRequest(http.MethodGet, "/", nil, map[string]string{"Accept": "application/json", "Accept-Language": "nl-BE"})
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("%v: %s", err, w.Body)
	}
	if got["locale"] != "en" || got["suggested"] != "nl" || got["legacy"] != nil {
		t.Errorf("got %v", got)
	}
}

func TestLegacyLocales_Unchanged(t *testing.T) {
	app := NewTestApp(t, TestConfig{})
	app.Router().GetFunc("/about", "pages", "about", func(r *http.Request) (any, error) {
		return map[string]any{
			"lang":      i18n.LanguageFromContext(r.Context()),
			"locale":    ViewVar(r, "locale"),
			"canonical": ViewVar(r, "canonical_url"),
			"fr":        ViewVar(r, "canonical_fr"),
			"cart":      LocalizePath(r, "/cart"),
		}, nil
	})

	w := app.PerformRequest(http.MethodGet, "/about?locale=fr", nil, map[string]string{"Accept": "application/json"})
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("%v: %s", err, w.Body)
	}
	want := map[string]any{"lang": "fr", "locale": "fr", "canonical": "/about?locale=fr", "fr": "/about?locale=fr", "cart": "/cart"}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %v, want %v", k, got[k], v)
		}
	}

	// Without an i18n block, a locale prefix is just a path segment
	if w := app.Get("/fr/about"); w.Code != http.StatusNotFound {
		t.Errorf("/fr/about: status %d, want 404", w.Code)
	}
}

func TestConfig_BaseURL(t *testing.T) {
	tests := []struct {
		conf Config
		want string
	}{
		{Config{BaseDomain: "floralynx.com", Secure: true}, "https://floralynx.com"},
		{Config{BaseDomain: "app.localhost:8080"}, "http://app.localhost:8080"},
		{Config{BaseDomain: "x.com", I18n: &i18n.Config{BaseURL: "https://www.x.com/"}}, "https://www.x.com"},
		{Config{}, ""},
	}
	for _, tt := range tests {
		if got := tt.conf.BaseURL(); got != tt.want {
			t.Errorf("%+v: %q, want %q", tt.conf, got, tt.want)
		}
	}
}

func TestPathLocales_TemplateHelpers(t *testing.T) {
	tpl := template.Must(template.New("test").Funcs(newTemplates("").fns).Parse(
		`{{ define "page" }}<a href="{{ localize "/cart" }}">{{ canonical_url }}</a>{{ alternate_links }}` +
			`{{ range alternates }}[{{ .Locale }}{{ if .Current }}*{{ end }}]{{ end }}{{ end }}`))
	a := &action{template: tpl, translator: i18n.NewTranslator("en")}

	mw := i18n.NewMiddleware(a.translator, i18n.Config{Locales: []string{"en", "nl"}, Strategy: i18n.StrategyPath, BaseURL: "https://example.com"})
	h, err := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := a.renderHTML(r, w, &Response{StatusCode: http.StatusOK}); err != nil {
			t.Fatalf("render: %v", err)
		}
	}))
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/nl/about", nil))
	body := w.Body.String()

	for _, want := range []string{
		`<a href="/nl/cart">https://example.com/nl/about</a>`,
		`<link rel="alternate" hreflang="en" href="https://example.com/about">`,
		`<link rel="alternate" hreflang="nl" href="https://example.com/nl/about">`,
		`<link rel="alternate" hreflang="x-default" href="https://example.com/about">`,
		`[en][nl*]`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in:\n%s", want, body)
		}
	}
}
