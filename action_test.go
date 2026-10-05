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
