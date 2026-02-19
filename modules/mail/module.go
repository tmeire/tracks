package mail

import (
	"encoding/json"
	"log/slog"

	"github.com/tmeire/tracks"
)

var (
	globalDriver    Driver
	globalTemplates *tracks.Templates
	defaultFrom     string
)

func constructDriver(config tracks.Config) Driver {
	confRaw, ok := config.Modules["mail"]
	if !ok {
		slog.Error("no mail configuration found")
		return NewLogDriver()
	}

	var conf Config
	if err := json.Unmarshal(confRaw, &conf); err != nil {
		slog.Error("failed to unmarshal mail configuration", "error", err)
		return NewLogDriver()
	}

	defaultFrom = conf.Defaults.From

	factory, ok := drivers[conf.DeliveryMethod]
	if !ok {
		slog.Error("mail config contained config for unknown delivery method", "method", conf.DeliveryMethod)
		return NewLogDriver()
	}
	d, err := factory(confRaw)
	if err != nil {
		slog.Error("failed to initialize mail driver", "method", conf.DeliveryMethod, "error", err)
		return NewLogDriver()
	}
	return d
}

// Register initializes the mail module with the given router's configuration.
func Register(r tracks.Router) tracks.Router {
	globalTemplates = r.Templates()

	// Configure from router config
	globalDriver = constructDriver(r.Config())

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
