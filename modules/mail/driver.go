package mail

import (
	"context"
	"encoding/json"
)

// Driver is the interface that every mail delivery backend must implement
type Driver interface {
	Send(ctx context.Context, msg *Message) error
}

// DriverFactory is a function that creates a Driver from raw JSON configuration
type DriverFactory func(conf json.RawMessage) (Driver, error)

var drivers = make(map[string]DriverFactory)

// RegisterDriver adds a new driver factory to the registry.
// This is typically called from a driver's init() function.
func RegisterDriver(name string, factory DriverFactory) {
	drivers[name] = factory
}
