package mail

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/tmeire/tracks"
	"github.com/tmeire/tracks/database"
)

type mockRouter struct {
	config tracks.Config
}

func (m *mockRouter) Clone() tracks.Router                                                     { return m }
func (m *mockRouter) Secure() bool                                                             { return false }
func (m *mockRouter) BaseDomain() string                                                       { return "" }
func (m *mockRouter) Database() database.Database                                             { return nil }
func (m *mockRouter) Module(mod tracks.Module) tracks.Router                                   { return mod(m) }
func (m *mockRouter) GlobalMiddleware(mw tracks.Middleware) tracks.Router                      { return m }
func (m *mockRouter) RequestMiddleware(mw tracks.Middleware) tracks.Router                     { return m }
func (m *mockRouter) Func(name string, fn any) tracks.Router                                   { return m }
func (m *mockRouter) Views(path string) tracks.Router                                          { return m }
func (m *mockRouter) Page(path, view string) tracks.Router                                     { return m }
func (m *mockRouter) Redirect(origin, destination string) tracks.Router                        { return m }
func (m *mockRouter) Serve(a tracks.Action) tracks.Router                                      { return m }
func (m *mockRouter) Controller(c tracks.Controller) tracks.Router                             { return m }
func (m *mockRouter) ControllerAtPath(path string, c tracks.Controller) tracks.Router          { return m }
func (m *mockRouter) Get(path, c, a string, rc tracks.ActionController, mws ...tracks.MiddlewareBuilder) tracks.Router {
	return m
}
func (m *mockRouter) GetFunc(path, c, a string, af tracks.ActionFunc, mws ...tracks.MiddlewareBuilder) tracks.Router {
	return m
}
func (m *mockRouter) PostFunc(path, c, a string, af tracks.ActionFunc, mws ...tracks.MiddlewareBuilder) tracks.Router {
	return m
}
func (m *mockRouter) PutFunc(path, c, a string, af tracks.ActionFunc, mws ...tracks.MiddlewareBuilder) tracks.Router {
	return m
}
func (m *mockRouter) PatchFunc(path, c, a string, af tracks.ActionFunc, mws ...tracks.MiddlewareBuilder) tracks.Router {
	return m
}
func (m *mockRouter) DeleteFunc(path, c, a string, af tracks.ActionFunc, mws ...tracks.MiddlewareBuilder) tracks.Router {
	return m
}
func (m *mockRouter) Resource(r tracks.Resource, mws ...tracks.MiddlewareBuilder) tracks.Router { return m }
func (m *mockRouter) ResourceAtPath(path string, r tracks.Resource, mws ...tracks.MiddlewareBuilder) tracks.Router {
	return m
}
func (m *mockRouter) Templates() *tracks.Templates { return nil }
func (m *mockRouter) Config() tracks.Config       { return m.config }
func (m *mockRouter) Handler() (http.Handler, error) { return nil, nil }
func (m *mockRouter) Run(ctx context.Context) error  { return nil }

func TestRegister(t *testing.T) {
	// Save current driver and restore it after tests
	oldDriver := globalDriver
	defer func() { globalDriver = oldDriver }()

	t.Run("defaults to log driver", func(t *testing.T) {
		globalDriver = nil
		r := &mockRouter{config: tracks.Config{}}
		Register(r)

		if _, ok := globalDriver.(*LogDriver); !ok {
			t.Errorf("expected LogDriver, got %T", globalDriver)
		}
	})

	t.Run("configures smtp driver", func(t *testing.T) {
		globalDriver = nil
		mailConf := Config{
			DeliveryMethod: "smtp",
			SMTP: SMTPConfig{
				Address: "localhost",
				Port:    25,
			},
		}
		raw, _ := json.Marshal(mailConf)
		
		r := &mockRouter{
			config: tracks.Config{
				Modules: map[string]json.RawMessage{
					"mail": raw,
				},
			},
		}
		Register(r)

		if _, ok := globalDriver.(*SMTPDriver); !ok {
			t.Errorf("expected SMTPDriver, got %T", globalDriver)
		}
	})

	t.Run("configures custom driver", func(t *testing.T) {
		globalDriver = nil
		called := false
		RegisterDriver("custom", func(conf json.RawMessage) (Driver, error) {
			called = true
			return &LogDriver{}, nil // reuse log driver as a mock
		})

		r := &mockRouter{
			config: tracks.Config{
				Modules: map[string]json.RawMessage{
					"mail": json.RawMessage(`{"delivery_method": "custom"}`),
				},
			},
		}
		Register(r)

		if !called {
			t.Error("expected custom driver factory to be called")
		}
	})
}
