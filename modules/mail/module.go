package mail

import (
	"github.com/tmeire/tracks"
)

var (
	globalDriver    Driver
	globalTemplates *tracks.Templates
	defaultFrom     string
)

// Register initializes the mail module with the given router's configuration.
func Register(r tracks.Router) tracks.Router {
	globalTemplates = r.Templates()

	// Default to log driver if nothing else is configured
	if globalDriver == nil {
		globalDriver = NewLogDriver()
	}

	return r
}

// SetDriver manually sets the global mail driver
func SetDriver(d Driver) {
	globalDriver = d
}

// NewMailer returns a base Mailer initialized with the global state
func NewMailer() Mailer {
	return Mailer{
		driver:    globalDriver,
		templates: globalTemplates,
		From:      defaultFrom,
	}
}
