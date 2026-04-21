package tracks

import (
	"context"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestRouter_Clone_Templates(t *testing.T) {
	ctx := context.Background()
	r := New(ctx)
	
	// Set initial basedir
	r.Views("./original_views")
	
	// Clone the router
	cloned := r.Clone()
	
	// Change basedir on the clone
	cloned.Views("./cloned_views")
	
	// Verify that the original router's basedir was NOT changed
	assert.Equal(t, "./original_views", r.Templates().basedir)
	assert.Equal(t, "./cloned_views", cloned.Templates().basedir)
}
