package tracks

import (
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/tmeire/tracks/database"
	"github.com/tmeire/tracks/database/sqlite"
)

type DomainDBConfig struct {
	DataDir    string
	Driver     string
	SchemaInit func(*sql.DB) error
}

type domainDBRouter struct {
	config DomainDBConfig
	dbs    map[string]database.Database
	mu     sync.RWMutex
}

func (r *domainDBRouter) ServeHTTP(next http.Handler) (http.Handler, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		domain := DomainFromContext(req.Context())
		if domain == "" {
			// Fallback to default database if no domain context
			next.ServeHTTP(w, req)
			return
		}

		db, err := r.getDB(domain)
		if err != nil {
			http.Error(w, "Failed to connect to domain database", http.StatusInternalServerError)
			return
		}

		ctx := database.WithDB(req.Context(), db)
		next.ServeHTTP(w, req.WithContext(ctx))
	}), nil
}

func (r *domainDBRouter) getDB(domain string) (database.Database, error) {
	r.mu.RLock()
	db, ok := r.dbs[domain]
	r.mu.RUnlock()
	if ok {
		return db, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double check
	if db, ok := r.dbs[domain]; ok {
		return db, nil
	}

	// Sanitize domain for filename
	safeName := sanitizeDomain(domain)
	dbPath := filepath.Join(r.config.DataDir, safeName+".db")

	// Create database
	sqlDB, err := sqlite.New(dbPath)
	if err != nil {
		return nil, err
	}

	// Initialize schema if needed
	if r.config.SchemaInit != nil {
		// Check if file was just created
		fi, err := os.Stat(dbPath)
		if err == nil && fi.Size() == 0 {
			if err := r.config.SchemaInit(sqlDB); err != nil {
				sqlDB.Close()
				return nil, err
			}
		} else if err == nil {
			// Even if not 0, we might want to run it, but usually migrations handle this.
			// For lightweight, we might just run it and let it fail if tables exist.
			if err := r.config.SchemaInit(sqlDB); err != nil {
				// Ignore errors from "table already exists" if we don't have migrations
			}
		}
	}

	r.dbs[domain] = sqlDB
	return sqlDB, nil
}

var domainSanitizeRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)

func sanitizeDomain(domain string) string {
	return domainSanitizeRegex.ReplaceAllString(domain, "_")
}

// DomainDatabase returns a middleware that routes database connections based on the domain.
func DomainDatabase(config DomainDBConfig) Middleware {
	dr := &domainDBRouter{
		config: config,
		dbs:    make(map[string]database.Database),
	}
	return dr.ServeHTTP
}
