package authentication

import (
	"context"
	"log/slog"
)

// PostUserCreationHook is a function that is executed after a user is successfully created.
type PostUserCreationHook func(ctx context.Context, user *User) error

var postUserCreationHooks []PostUserCreationHook

// OnUserCreated registers a callback to be executed after a user is created.
func OnUserCreated(hook PostUserCreationHook) {
	postUserCreationHooks = append(postUserCreationHooks, hook)
}

// executePostUserCreationHooks triggers all registered post-user-creation hooks.
func executePostUserCreationHooks(ctx context.Context, user *User) error {
	for _, hook := range postUserCreationHooks {
		if err := hook(ctx, user); err != nil {
			// Log the error but allow the process to continue as suggested in the proposal.
			slog.Error("post-user-creation hook failed", "user_id", user.ID, "error", err)
		}
	}
	return nil
}
