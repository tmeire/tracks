package tracks

import (
	"net/http"

	"github.com/tmeire/tracks/i18n"
)

// LocalizePath returns path in the language of the request, e.g. "/about" becomes "/nl/about"
// on a Dutch page when the path locale strategy is enabled. Otherwise path is returned as is.
func LocalizePath(r *http.Request, path string) string {
	return i18n.LocalizePath(r.Context(), path)
}

// SetAlternates declares the language versions of the current page as locale -> unprefixed path.
// Locales left out get no alternate link. Without a call, every configured locale serves the
// same path.
//
//	tracks.SetAlternates(r, map[string]string{"en": "/coloring-pages/cow", "nl": "/kleurplaten/koe"})
func SetAlternates(r *http.Request, alternates map[string]string) {
	i18n.SetAlternates(r.Context(), alternates)
}

// T translates key in the language of the request.
func T(r *http.Request, key string, params ...any) string {
	return i18n.T(r.Context(), key, params...)
}
