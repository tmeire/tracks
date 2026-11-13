package tracks

import (
	"errors"
	"html/template"
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

type testTemplate struct{ *template.Template }

func (t testTemplate) Clone() (*template.Template, error) { return t.Template.Clone() }

func TestAction_ErrorTemplateStatusReachesClient(t *testing.T) {
	tpl := template.Must(template.New("root").Parse(`{{define "404"}}<h1>not found</h1>{{end}}{{define "page"}}<p>ok</p>{{end}}`))

	run := func(resp *Response, tplToUse Template) *httptest.ResponseRecorder {
		a := &action{name: "pages#show", template: tplToUse, impl: func(r *http.Request) (any, error) { return resp, nil }}
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Accept", "text/html")
		rec := httptest.NewRecorder()
		a.ServeHTTP(rec, req)
		return rec
	}

	t.Run("html error template keeps its status", func(t *testing.T) {
		rec := run(&Response{StatusCode: http.StatusNotFound}, testTemplate{tpl})
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "<h1>not found</h1>", rec.Body.String())
		assert.Equal(t, "text/html", rec.Header().Get("Content-Type"))
	})

	t.Run("missing error template falls back to json with the status", func(t *testing.T) {
		rec := run(&Response{StatusCode: http.StatusTeapot, Data: map[string]string{"a": "b"}}, testTemplate{tpl})
		assert.Equal(t, http.StatusTeapot, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
	})

	t.Run("success still renders the page with 200", func(t *testing.T) {
		tpl2 := template.Must(template.New("root").Parse(`{{define "page"}}<p>ok</p>{{end}}`))
		rec := run(&Response{}, testTemplate{tpl2})
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "<p>ok</p>", rec.Body.String())
	})

	t.Run("redirect is untouched", func(t *testing.T) {
		rec := run(&Response{Location: "/y"}, testTemplate{tpl})
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/y", rec.Header().Get("Location"))
	})
}
