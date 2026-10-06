package i18n

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// Locale strategies supported by Config.Strategy.
const (
	// StrategyDetect picks the language from the ?locale= query parameter, the locale cookie,
	// the session or the Accept-Language header, in that order. URLs are shared by all languages.
	StrategyDetect = "detect"
	// StrategyPath picks the language from the first path segment (/nl/...). The default
	// language is served without a prefix. Cookies and headers never change the language.
	StrategyPath = "path"
)

// Config configures the supported locales and how the request language is resolved.
// It maps to the "i18n" block in config.json:
//
//	"i18n": {"default": "en", "locales": ["en", "nl"], "strategy": "path"}
type Config struct {
	// Default is the fallback language, served without a path prefix. Defaults to "en".
	Default string `json:"default"`
	// Locales lists every supported language, including the default.
	Locales []string `json:"locales"`
	// Strategy is StrategyDetect (default) or StrategyPath.
	Strategy string `json:"strategy"`
	// BaseURL is the absolute origin used for canonical and alternate URLs, e.g. https://example.com.
	// When empty, tracks derives it from base_domain and secure.
	BaseURL string `json:"base_url"`
}

// Normalize fills in defaults: "en" as default language, the default language in Locales,
// StrategyDetect as strategy and no trailing slash on BaseURL.
func (c Config) Normalize() Config {
	if c.Default == "" {
		c.Default = "en"
	}
	if c.Strategy == "" {
		c.Strategy = StrategyDetect
	}
	if !c.Supports(c.Default) {
		c.Locales = append([]string{c.Default}, c.Locales...)
	}
	c.BaseURL = strings.TrimSuffix(c.BaseURL, "/")
	return c
}

// Validate reports configuration errors such as an unknown strategy.
func (c Config) Validate() error {
	if c.Strategy != StrategyDetect && c.Strategy != StrategyPath {
		return fmt.Errorf("i18n: unknown strategy %q, expected %q or %q", c.Strategy, StrategyDetect, StrategyPath)
	}
	for _, l := range c.Locales {
		if l == "" || strings.Contains(l, "/") {
			return fmt.Errorf("i18n: invalid locale %q", l)
		}
	}
	return nil
}

// Supports reports whether locale is one of the configured locales.
func (c Config) Supports(locale string) bool {
	for _, l := range c.Locales {
		if l == locale {
			return true
		}
	}
	return false
}

// splitPrefix returns the locale in the first segment of path and the remaining path,
// e.g. "/nl/about" -> ("nl", "/about", true). It only matches configured locales.
func (c Config) splitPrefix(path string) (locale, rest string, ok bool) {
	if !strings.HasPrefix(path, "/") {
		return "", path, false
	}
	seg, rest, _ := strings.Cut(path[1:], "/")
	if !c.Supports(seg) {
		return "", path, false
	}
	return seg, "/" + rest, true
}

// localize returns the path for locale. Paths that are not root-relative, already carry
// a locale prefix or target the default locale are returned unchanged.
func (c Config) localize(locale, path string) string {
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return path
	}
	if locale == c.Default || !c.Supports(locale) {
		return path
	}
	// Only inspect the path part: "/nl?x=1" and "/nl#top" are already localized too.
	p := path
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	if _, _, ok := c.splitPrefix(p); ok {
		return path
	}
	if p == "/" {
		return "/" + locale + path[1:]
	}
	return "/" + locale + path
}

// alternatePath returns the path that serves path in locale under the configured strategy.
func (c Config) alternatePath(locale, path string) string {
	if c.Strategy == StrategyPath {
		return c.localize(locale, path)
	}
	if locale == c.Default || !strings.HasPrefix(path, "/") {
		return path
	}
	return path + "?locale=" + locale
}

func (c Config) absolute(path string) string {
	if strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "//") {
		return c.BaseURL + path
	}
	return path
}

type stateKey struct{}

// requestState is shared by pointer through the request context so that handlers can
// override the alternates after the middleware ran.
type requestState struct {
	cfg        Config
	lang       string
	path       string // request path without locale prefix
	suggested  string
	alternates map[string]string // nil means "same path in every locale"
}

func stateFromContext(ctx context.Context) *requestState {
	st, _ := ctx.Value(stateKey{}).(*requestState)
	return st
}

// ConfigFromContext returns the i18n configuration of the current request, if any.
func ConfigFromContext(ctx context.Context) (Config, bool) {
	if st := stateFromContext(ctx); st != nil {
		return st.cfg, true
	}
	return Config{}, false
}

// NewMiddleware resolves the request language according to cfg.
//
// With StrategyPath, a configured locale prefix is stripped from the URL before routing,
// so every route serves all languages: /nl/about is routed as /about with language "nl".
// Requests prefixed with the default locale are redirected (301) to the unprefixed URL.
func NewMiddleware(t *Translator, cfg Config) func(next http.Handler) (http.Handler, error) {
	cfg = cfg.Normalize()
	return func(next http.Handler) (http.Handler, error) {
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lang := cfg.Default
			suggested := ""

			if cfg.Strategy == StrategyPath {
				if locale, rest, ok := cfg.splitPrefix(r.URL.Path); ok {
					if locale == cfg.Default && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
						target := rest
						if r.URL.RawQuery != "" {
							target += "?" + r.URL.RawQuery
						}
						http.Redirect(w, r, target, http.StatusMovedPermanently)
						return
					}
					lang = locale
					r = stripPrefix(r, "/"+locale, rest)
				}
				if pref := PreferredLanguage(r, cfg); pref != "" && pref != lang {
					suggested = pref
				}
			} else if detected := DetectLanguage(r, cfg.Default); cfg.Supports(detected) {
				lang = detected
			}

			st := &requestState{cfg: cfg, lang: lang, path: r.URL.Path, suggested: suggested}
			ctx := context.WithValue(r.Context(), stateKey{}, st)
			ctx = WithLanguage(ctx, lang)
			ctx = WithTranslator(ctx, t)
			next.ServeHTTP(w, r.WithContext(ctx))
		}), nil
	}
}

// stripPrefix returns a shallow copy of r whose URL no longer carries prefix.
func stripPrefix(r *http.Request, prefix, rest string) *http.Request {
	r2 := new(http.Request)
	*r2 = *r
	u := *r.URL
	u.Path = rest
	if u.RawPath != "" {
		u.RawPath = strings.TrimPrefix(u.RawPath, prefix)
		if u.RawPath == "" {
			u.RawPath = "/"
		}
	}
	r2.URL = &u
	return r2
}

// PreferredLanguage returns the supported locale the visitor prefers, based on the locale
// cookie and the Accept-Language header, or "" when none of them matches a configured locale.
func PreferredLanguage(r *http.Request, cfg Config) string {
	if c, err := r.Cookie("locale"); err == nil && cfg.Supports(c.Value) {
		return c.Value
	}

	type candidate struct {
		tag string
		q   float64
	}
	var candidates []candidate
	for _, part := range strings.Split(r.Header.Get("Accept-Language"), ",") {
		tag, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		if tag == "" {
			continue
		}
		q := 1.0
		if v, ok := strings.CutPrefix(strings.TrimSpace(params), "q="); ok {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				q = f
			}
		}
		candidates = append(candidates, candidate{strings.ToLower(tag), q})
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].q > candidates[j].q })

	for _, c := range candidates {
		if c.q <= 0 {
			continue
		}
		if cfg.Supports(c.tag) {
			return c.tag
		}
		if base, _, _ := strings.Cut(c.tag, "-"); cfg.Supports(base) {
			return base
		}
	}
	return ""
}

// SuggestedLanguage returns the supported language the visitor prefers when it differs from
// the language of the current page (path strategy only). Use it to offer a language switch;
// never to redirect.
func SuggestedLanguage(ctx context.Context) string {
	if st := stateFromContext(ctx); st != nil {
		return st.suggested
	}
	return ""
}

// LocalizePath returns path in the language of the current request, e.g. "/about" becomes
// "/nl/about" on a Dutch page. It is a no-op unless the path strategy is enabled.
func LocalizePath(ctx context.Context, path string) string {
	return LocalizePathFor(ctx, LanguageFromContext(ctx), path)
}

// LocalizePathFor returns path in the given locale. It is a no-op unless the path strategy is enabled.
func LocalizePathFor(ctx context.Context, locale, path string) string {
	st := stateFromContext(ctx)
	if st == nil || st.cfg.Strategy != StrategyPath {
		return path
	}
	return st.cfg.localize(locale, path)
}

// URLFor returns the absolute URL of path in the given locale. Redirects to absolute URLs are
// never localized, so use it to send a visitor to another language than the current one.
func URLFor(ctx context.Context, locale, path string) string {
	st := stateFromContext(ctx)
	if st == nil {
		return path
	}
	return st.cfg.absolute(st.cfg.alternatePath(locale, path))
}

// SetAlternates declares which locales the current page exists in, mapping each locale to its
// unprefixed path. Use it when paths differ per language (translated slugs) or when a page is
// not available in every language; locales missing from the map get no alternate link.
//
//	i18n.SetAlternates(ctx, map[string]string{
//		"en": "/coloring-pages/cow",
//		"nl": "/kleurplaten/koe", // served as /nl/kleurplaten/koe
//	})
func SetAlternates(ctx context.Context, alternates map[string]string) {
	st := stateFromContext(ctx)
	if st == nil {
		return
	}
	st.alternates = make(map[string]string, len(alternates))
	for locale, path := range alternates {
		st.alternates[locale] = path
	}
}

// Alternate is one language version of the current page.
type Alternate struct {
	Locale  string
	Path    string // path relative to the site root, including the locale prefix
	URL     string // absolute URL
	Current bool   // true for the language of the current request
}

// Alternates returns the language versions of the current page, in the order of Config.Locales.
func Alternates(ctx context.Context) []Alternate {
	st := stateFromContext(ctx)
	if st == nil {
		return nil
	}
	alts := make([]Alternate, 0, len(st.cfg.Locales))
	for _, locale := range st.cfg.Locales {
		path := st.path
		if st.alternates != nil {
			p, ok := st.alternates[locale]
			if !ok {
				continue
			}
			path = p
		}
		path = st.cfg.alternatePath(locale, path)
		alts = append(alts, Alternate{
			Locale:  locale,
			Path:    path,
			URL:     st.cfg.absolute(path),
			Current: locale == st.lang,
		})
	}
	return alts
}

// CanonicalURL returns the absolute URL of the current page in the current language,
// without query parameters.
func CanonicalURL(ctx context.Context) string {
	st := stateFromContext(ctx)
	if st == nil {
		return ""
	}
	path := st.path
	if p, ok := st.alternates[st.lang]; ok {
		path = p
	}
	return st.cfg.absolute(st.cfg.alternatePath(st.lang, path))
}

// AlternateLinks renders <link rel="alternate" hreflang="..."> tags for every language version
// of the current page, plus x-default pointing to the default language. It renders nothing
// when the page exists in a single language.
func AlternateLinks(ctx context.Context) template.HTML {
	st := stateFromContext(ctx)
	alts := Alternates(ctx)
	if st == nil || len(alts) < 2 {
		return ""
	}
	var b strings.Builder
	for _, a := range alts {
		fmt.Fprintf(&b, "<link rel=\"alternate\" hreflang=\"%s\" href=\"%s\">\n",
			template.HTMLEscapeString(a.Locale), template.HTMLEscapeString(a.URL))
	}
	for _, a := range alts {
		if a.Locale == st.cfg.Default {
			fmt.Fprintf(&b, "<link rel=\"alternate\" hreflang=\"x-default\" href=\"%s\">\n", template.HTMLEscapeString(a.URL))
		}
	}
	return template.HTML(b.String())
}
