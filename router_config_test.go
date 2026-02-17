package tracks

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouter_Config(t *testing.T) {
	conf := Config{
		Name:       "TestApp",
		Version:    "1.0.0",
		BaseDomain: "test.local",
		Modules: map[string]json.RawMessage{
			"test": json.RawMessage(`{"key": "value"}`),
		},
	}

	r := &router{
		config:             conf,
		requestMiddlewares: &middlewares{},
	}

	assert.Equal(t, conf, r.Config())

	t.Run("Clone preserves config", func(t *testing.T) {
		cloned := r.Clone()
		assert.Equal(t, conf, cloned.Config())
	})

	t.Run("errRouter returns empty config", func(t *testing.T) {
		er := errRouter{}
		assert.Equal(t, Config{}, er.Config())
	})
}
