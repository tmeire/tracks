package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/tmeire/tracks"
	"github.com/tmeire/tracks/modules/multitenancy"
)

type Config struct {
	Provider string          `json:"provider"`
	Config   json.RawMessage `json:"config"`
}

var globalService *StorageService

type storageKey struct{}
type centralDBKey struct{}

// constructDriver creates the appropriate storage driver from the configuration.
func constructDriver(config tracks.Config) (Driver, error) {
	confRaw, ok := config.Modules["storage"]
	if !ok {
		return nil, fmt.Errorf("no storage configuration found")
	}

	var conf Config
	if err := json.Unmarshal(confRaw, &conf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal storage configuration: %w", err)
	}

	factory, ok := drivers[conf.Provider]
	if !ok {
		return nil, fmt.Errorf("storage config contained config for unknown provider %q", conf.Provider)
	}
	d, err := factory(conf.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage driver for provider %q: %w", conf.Provider, err)
	}
	return d, nil
}

// Register initializes the storage module with the given router's configuration.
// It sets up the storage driver, applies migrations, and registers a middleware
// that injects the StorageService into the request context.
func Register(r tracks.Router) tracks.Router {
	driver, err := constructDriver(r.Config())
	if err != nil {
		return tracks.NewErrorRouter(err)
	}

	// Initialize the storage service with a default multitenancy extractor
	service, err := NewStorageService(context.Background(), r.Database(), driver, func(ctx context.Context) (string, error) {
		return multitenancy.SubdomainFromContext(ctx), nil
	})
	if err != nil {
		return tracks.NewErrorRouter(fmt.Errorf("failed to initialize storage service: %w", err))
	}

	// Store globally for background jobs and non-request contexts
	globalService = service

	// Register middleware to inject the service and central DB into the context
	r.GlobalMiddleware(func(next http.Handler) (http.Handler, error) {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()
			ctx = context.WithValue(ctx, storageKey{}, service)
			ctx = context.WithValue(ctx, centralDBKey{}, service.CentralDB())
			next.ServeHTTP(w, req.WithContext(ctx))
		}), nil
	})

	return r
}

// FromContext returns the StorageService stored in the context, or nil if not found.
func FromContext(ctx context.Context) *StorageService {
	if s, ok := ctx.Value(storageKey{}).(*StorageService); ok {
		return s
	}
	return nil
}

// StorageServiceFromContext returns the globally registered StorageService.
// This is intended for use in background jobs where no request context is available.
func StorageServiceFromContext() *StorageService {
	return globalService
}
