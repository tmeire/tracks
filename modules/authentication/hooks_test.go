package authentication

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPostUserCreationHooks(t *testing.T) {
	// Reset hooks for testing
	postUserCreationHooks = nil

	var called bool
	var calledUser *User

	OnUserCreated(func(ctx context.Context, user *User) error {
		called = true
		calledUser = user
		return nil
	})

	user := &User{ID: "test@example.com", Email: "test@example.com", Name: "Test User"}
	err := executePostUserCreationHooks(context.Background(), user)

	assert.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, user, calledUser)
}

func TestPostUserCreationHooks_Multiple(t *testing.T) {
	// Reset hooks for testing
	postUserCreationHooks = nil

	var callCount int

	OnUserCreated(func(ctx context.Context, user *User) error {
		callCount++
		return nil
	})

	OnUserCreated(func(ctx context.Context, user *User) error {
		callCount++
		return nil
	})

	user := &User{ID: "test@example.com", Email: "test@example.com", Name: "Test User"}
	err := executePostUserCreationHooks(context.Background(), user)

	assert.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestPostUserCreationHooks_ContinueOnError(t *testing.T) {
	// Reset hooks for testing
	postUserCreationHooks = nil

	var calledAfter bool

	OnUserCreated(func(ctx context.Context, user *User) error {
		return assert.AnError
	})

	OnUserCreated(func(ctx context.Context, user *User) error {
		calledAfter = true
		return nil
	})

	user := &User{ID: "test@example.com", Email: "test@example.com", Name: "Test User"}
	err := executePostUserCreationHooks(context.Background(), user)

	assert.NoError(t, err)
	assert.True(t, calledAfter)
}
