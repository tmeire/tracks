package tracks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseController_Scheme(t *testing.T) {
	t.Run("uninitialized controller should not panic", func(t *testing.T) {
		bc := &BaseController{}
		// This used to panic because bc.router was nil
		scheme := bc.Scheme()
		assert.Equal(t, "http", scheme)
	})

	t.Run("initialized controller with secure config", func(t *testing.T) {
		bc := &BaseController{}
		r := &mockRouter{secure: true}
		bc.Inject(r)
		
		assert.Equal(t, "https", bc.Scheme())
	})

	t.Run("initialized controller with insecure config", func(t *testing.T) {
		bc := &BaseController{}
		r := &mockRouter{secure: false}
		bc.Inject(r)
		
		assert.Equal(t, "http", bc.Scheme())
	})
}

type mockRouter struct {
	Router
	secure bool
}

func (m *mockRouter) Secure() bool {
	return m.secure
}
