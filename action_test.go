package tracks

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAction_InternalErrorIsHiddenFromUser(t *testing.T) {
	a := &action{
		name: "sessions#create",
		impl: func(r *http.Request) (any, error) {
			return nil, errors.New("no such column: password")
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/sessions", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()

	a.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "An internal server error occurred")
	assert.NotContains(t, rec.Body.String(), "no such column")
}

func TestAction_RedirectStatus(t *testing.T) {
	tests := []struct {
		name   string
		resp   *Response
		htmx   bool
		status int
	}{
		{"default is see other", &Response{Location: "/cart"}, false, http.StatusSeeOther},
		{"non-redirect status falls back to see other", &Response{StatusCode: http.StatusOK, Location: "/cart"}, false, http.StatusSeeOther},
		{"moved permanently", &Response{StatusCode: http.StatusMovedPermanently, Location: "/new"}, false, http.StatusMovedPermanently},
		{"found", &Response{StatusCode: http.StatusFound, Location: "/new"}, false, http.StatusFound},
		{"permanent redirect", &Response{StatusCode: http.StatusPermanentRedirect, Location: "/new"}, false, http.StatusPermanentRedirect},
		{"htmx", &Response{StatusCode: http.StatusMovedPermanently, Location: "/new"}, true, http.StatusAccepted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &action{name: "pages#show", impl: func(r *http.Request) (any, error) { return tt.resp, nil }}
			req := httptest.NewRequest(http.MethodGet, "/old", nil)
			req.Header.Set("Accept", "text/html")
			if tt.htmx {
				req.Header.Set("HX-Request", "true")
			}
			rec := httptest.NewRecorder()

			a.ServeHTTP(rec, req)

			assert.Equal(t, tt.status, rec.Code)
			if tt.htmx {
				assert.Equal(t, tt.resp.Location, rec.Header().Get("HX-Redirect"))
			} else {
				assert.Equal(t, tt.resp.Location, rec.Header().Get("Location"))
			}
		})
	}
}
