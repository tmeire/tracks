package i18n

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var pathConfig = Config{
	Default:  "en",
	Locales:  []string{"en", "nl", "fr"},
	Strategy: StrategyPath,
	BaseURL:  "https://example.com",
}

type captured struct {
	path, rawPath, lang string
	req             *http.Request
}

// serve runs req through the middleware and records what the next handler saw.
func serve(t *testing.T, cfg Config, req *http.Request) (*httptest.ResponseRecorder, *captured) {
	t.Helper()
	got := &captured{}
	h, err := NewMiddleware(NewTranslator(cfg.Default), cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.path = r.URL.Path
		got.rawPath = r.URL.RawPath
		got.lang = LanguageFromContext(r.Context())
		got.req = r
	}))
	if err != nil {
		t.Fatalf("middleware: %v", err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w, got
}

func TestPathStrategy_StripsLocalePrefix(t *testing.T) {
	tests := []struct {
		url, wantPath, wantLang string
	}{
		{"/nl/coloring-pages/cow", "/coloring-pages/cow", "nl"},
		{"/nl", "/", "nl"},
		{"/nl/", "/", "nl"},
		{"/fr/a?x=1", "/a", "fr"},
		{"/coloring-pages", "/coloring-pages", "en"},
		{"/", "/", "en"},
		{"/nlx/a", "/nlx/a", "en"}, // not a locale segment
		{"/de/a", "/de/a", "en"},   // unsupported locale passes through (and 404s in the app)
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			_, got := serve(t, pathConfig, httptest.NewRequest(http.MethodGet, tt.url, nil))
			if got.path != tt.wantPath || got.lang != tt.wantLang {
				t.Errorf("got path %q lang %q, want %q %q", got.path, got.lang, tt.wantPath, tt.wantLang)
			}
		})
	}
}

func TestPathStrategy_StripsEscapedPath(t *testing.T) {
	_, got := serve(t, pathConfig, httptest.NewRequest(http.MethodGet, "/nl/a%2Fb", nil))
	if got.path != "/a/b" || got.rawPath != "/a%2Fb" {
		t.Errorf("got path %q raw %q", got.path, got.rawPath)
	}
}

func TestPathStrategy_DoesNotMutateOriginalRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/nl/a", nil)
	serve(t, pathConfig, req)
	if req.URL.Path != "/nl/a" {
		t.Errorf("original request path changed to %q", req.URL.Path)
	}
}

func TestPathStrategy_RedirectsDefaultLocalePrefix(t *testing.T) {
	w, got := serve(t, pathConfig, httptest.NewRequest(http.MethodGet, "/en/coloring-pages?page=2", nil))
	if w.Code != http.StatusMovedPermanently || w.Header().Get("Location") != "/coloring-pages?page=2" {
		t.Errorf("got %d %q", w.Code, w.Header().Get("Location"))
	}
	if got.req != nil {
		t.Error("next handler should not run")
	}

	// Non-idempotent requests are not redirected, only stripped
	_, got = serve(t, pathConfig, httptest.NewRequest(http.MethodPost, "/en/cart", nil))
	if got.path != "/cart" || got.lang != "en" {
		t.Errorf("POST: got path %q lang %q", got.path, got.lang)
	}
}

func TestPathStrategy_IgnoresCookiesAndHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/a?locale=fr", nil)
	req.Header.Set("Accept-Language", "nl-BE,nl;q=0.9")
	req.AddCookie(&http.Cookie{Name: "locale", Value: "fr"})
	_, got := serve(t, pathConfig, req)
	if got.lang != "en" {
		t.Errorf("lang = %q, want en", got.lang)
	}
	if s := SuggestedLanguage(got.req.Context()); s != "fr" {
		t.Errorf("suggested = %q, want fr (cookie wins)", s)
	}
}

func TestPathStrategy_SuggestsPreferredLanguage(t *testing.T) {
	tests := []struct {
		url, accept, want string
	}{
		{"/a", "nl-BE,nl;q=0.9,en;q=0.8", "nl"},
		{"/a", "de-DE,fr;q=0.5,nl;q=0.7", "nl"},
		{"/a", "de-DE", ""},
		{"/a", "en-US", ""},
		{"/nl/a", "nl", ""}, // already on the preferred language
		{"/nl/a", "en", "en"},
		{"/a", "nl;q=0", ""},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, tt.url, nil)
		req.Header.Set("Accept-Language", tt.accept)
		_, got := serve(t, pathConfig, req)
		if s := SuggestedLanguage(got.req.Context()); s != tt.want {
			t.Errorf("%s %q: suggested %q, want %q", tt.url, tt.accept, s, tt.want)
		}
	}
}

func TestDetectStrategy_OnlyAcceptsConfiguredLocales(t *testing.T) {
	cfg := Config{Locales: []string{"en", "fr"}}

	_, got := serve(t, cfg, httptest.NewRequest(http.MethodGet, "/a?locale=fr", nil))
	if got.lang != "fr" {
		t.Errorf("lang = %q, want fr", got.lang)
	}

	_, got = serve(t, cfg, httptest.NewRequest(http.MethodGet, "/a?locale=zz", nil))
	if got.lang != "en" {
		t.Errorf("lang = %q, want en", got.lang)
	}

	// The path is never rewritten in detect mode
	_, got = serve(t, cfg, httptest.NewRequest(http.MethodGet, "/fr/a", nil))
	if got.path != "/fr/a" || got.lang != "en" {
		t.Errorf("got path %q lang %q", got.path, got.lang)
	}
}

func TestLocalizePath(t *testing.T) {
	_, got := serve(t, pathConfig, httptest.NewRequest(http.MethodGet, "/nl/a", nil))
	ctx := got.req.Context()

	tests := []struct{ in, want string }{
		{"/cart", "/nl/cart"},
		{"/", "/nl"},
		{"/?q=1", "/nl?q=1"},
		{"/#top", "/nl#top"},
		{"/search?q=1", "/nl/search?q=1"},
		{"/nl/cart", "/nl/cart"},
		{"/nl", "/nl"},
		{"/fr/cart", "/fr/cart"},
		{"https://other.com/x", "https://other.com/x"},
		{"//cdn.example.com/x", "//cdn.example.com/x"},
		{"relative", "relative"},
	}
	for _, tt := range tests {
		if out := LocalizePath(ctx, tt.in); out != tt.want {
			t.Errorf("LocalizePath(%q) = %q, want %q", tt.in, out, tt.want)
		}
	}

	if out := LocalizePathFor(ctx, "en", "/cart"); out != "/cart" {
		t.Errorf("LocalizePathFor(en) = %q", out)
	}
	if out := URLFor(ctx, "en", "/cart"); out != "https://example.com/cart" {
		t.Errorf("URLFor(en) = %q", out)
	}
}

func TestLocalizePath_NoopOutsidePathStrategy(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/a", nil)
	if out := LocalizePath(req.Context(), "/cart"); out != "/cart" {
		t.Errorf("without middleware: %q", out)
	}
	_, got := serve(t, Config{Locales: []string{"en", "fr"}}, httptest.NewRequest(http.MethodGet, "/a?locale=fr", nil))
	if out := LocalizePath(got.req.Context(), "/cart"); out != "/cart" {
		t.Errorf("detect strategy: %q", out)
	}
}

func TestAlternates_SamePathInEveryLocale(t *testing.T) {
	_, got := serve(t, pathConfig, httptest.NewRequest(http.MethodGet, "/nl/about?utm=x", nil))
	ctx := got.req.Context()

	alts := Alternates(ctx)
	want := []Alternate{
		{Locale: "en", Path: "/about", URL: "https://example.com/about"},
		{Locale: "nl", Path: "/nl/about", URL: "https://example.com/nl/about", Current: true},
		{Locale: "fr", Path: "/fr/about", URL: "https://example.com/fr/about"},
	}
	if len(alts) != len(want) {
		t.Fatalf("got %d alternates: %+v", len(alts), alts)
	}
	for i := range want {
		if alts[i] != want[i] {
			t.Errorf("alternate %d = %+v, want %+v", i, alts[i], want[i])
		}
	}
	if c := CanonicalURL(ctx); c != "https://example.com/nl/about" {
		t.Errorf("canonical = %q", c)
	}
}

func TestSetAlternates_TranslatedSlugsAndMissingLocales(t *testing.T) {
	_, got := serve(t, pathConfig, httptest.NewRequest(http.MethodGet, "/nl/kleurplaten/koe", nil))
	ctx := got.req.Context()

	SetAlternates(ctx, map[string]string{
		"en": "/coloring-pages/cow",
		"nl": "/kleurplaten/koe",
	})

	links := string(AlternateLinks(ctx))
	want := `<link rel="alternate" hreflang="en" href="https://example.com/coloring-pages/cow">
<link rel="alternate" hreflang="nl" href="https://example.com/nl/kleurplaten/koe">
<link rel="alternate" hreflang="x-default" href="https://example.com/coloring-pages/cow">
`
	if links != want {
		t.Errorf("links:\n%s\nwant:\n%s", links, want)
	}
	if strings.Contains(links, "hreflang=\"fr\"") {
		t.Error("fr has no translation and must not be listed")
	}
	if c := CanonicalURL(ctx); c != "https://example.com/nl/kleurplaten/koe" {
		t.Errorf("canonical = %q", c)
	}
}

func TestAlternateLinks_SingleLanguageRendersNothing(t *testing.T) {
	_, got := serve(t, pathConfig, httptest.NewRequest(http.MethodGet, "/a", nil))
	SetAlternates(got.req.Context(), map[string]string{"en": "/a"})
	if links := AlternateLinks(got.req.Context()); links != "" {
		t.Errorf("got %q", links)
	}
}

func TestAlternates_DetectStrategyUsesQueryParameter(t *testing.T) {
	cfg := Config{Locales: []string{"en", "fr"}, BaseURL: "https://example.com/"}
	_, got := serve(t, cfg, httptest.NewRequest(http.MethodGet, "/a?locale=fr", nil))
	alts := Alternates(got.req.Context())
	if len(alts) != 2 || alts[0].URL != "https://example.com/a" || alts[1].URL != "https://example.com/a?locale=fr" || !alts[1].Current {
		t.Errorf("got %+v", alts)
	}
}

func TestConfig_NormalizeAndValidate(t *testing.T) {
	cfg := Config{Locales: []string{"nl"}}.Normalize()
	if cfg.Default != "en" || cfg.Strategy != StrategyDetect || len(cfg.Locales) != 2 || cfg.Locales[0] != "en" {
		t.Errorf("normalized = %+v", cfg)
	}
	if err := (Config{Strategy: "subdomain"}).Normalize().Validate(); err == nil {
		t.Error("expected error for unknown strategy")
	}
	if _, err := NewMiddleware(NewTranslator("en"), Config{Strategy: "subdomain"})(http.NotFoundHandler()); err == nil {
		t.Error("middleware should refuse an invalid config")
	}
}

func TestTranslate_LogsMissingKeysOnce(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "en.json"), []byte(`{"hello":"Hello","only_en":"English"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "nl.json"), []byte(`{"hello":"Hallo"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	tr := NewTranslator("en")
	if err := tr.LoadTranslations(dir); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(prev)

	if got := tr.Translate("nl", "only_en"); got != "English" {
		t.Errorf("fallback = %q", got)
	}
	tr.Translate("nl", "only_en")
	tr.Translate("nl", "hello")
	tr.Translate("en", "nope")
	tr.Translate("zz", "hello") // unknown languages are not logged

	logs := buf.String()
	if n := strings.Count(logs, "missing translation"); n != 2 {
		t.Errorf("expected 2 warnings, got %d:\n%s", n, logs)
	}
	if !strings.Contains(logs, "locale=nl key=only_en") || !strings.Contains(logs, "locale=en key=nope") {
		t.Errorf("unexpected logs:\n%s", logs)
	}
}
