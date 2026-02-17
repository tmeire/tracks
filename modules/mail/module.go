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

// Register initializes the mail module with the given router's configuration.
func Register(r tracks.Router) tracks.Router {
	globalTemplates = r.Templates()

	// Configure from router config
	if confRaw, ok := r.Config().Modules["mail"]; ok {
		var conf Config
		if err := json.Unmarshal(confRaw, &conf); err != nil {
			slog.Error("failed to unmarshal mail configuration", "error", err)
		} else {
			defaultFrom = conf.Defaults.From

			if factory, ok := drivers[conf.DeliveryMethod]; ok {
				var err error
				globalDriver, err = factory(confRaw)
				if err != nil {
					slog.Error("failed to initialize mail driver", "method", conf.DeliveryMethod, "error", err)
				}
			} else if conf.DeliveryMethod != "" {
				slog.Warn("unknown mail delivery method, defaulting to log", "method", conf.DeliveryMethod)
			}
		}
	}

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
