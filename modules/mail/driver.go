package mail

import (
	"context"
)

// Driver is the interface that every mail delivery backend must implement
type Driver interface {
	Send(ctx context.Context, msg *Message) error
}
