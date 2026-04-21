# Changelog

## [v0.0.50] - 2026-04-22
### Fixed
- Router: Cloned routers no longer inherit global middlewares by default. This fixes a critical bug where tenant-specific database connections were being overwritten by the central database connection from the main router during subdomain requests.

## [v0.0.49] - 2026-04-22
### Added
- Router: Added `SkipDefaultMiddlewares()` to the `Router` interface to allow sub-routers (clones) to opt-out of inherited global middlewares.
### Fixed
- Multitenancy: Use `SkipDefaultMiddlewares()` for tenant-specific sub-routers to prevent the default central database middleware from overwriting the tenant-specific database connection.

## [v0.0.48] - 2026-04-22
### Added
- OpenTelemetry: Added `stdouttrace` exporter to log traces to stdout in addition to the OTLP gRPC exporter.

## [v0.0.47] - 2026-04-21
### Fixed
- **Critical:** Fixed a context-loss bug in the multitenancy `splitter` middleware. The central database connection was being dropped when updating view variables, leading to nil-pointer panics in role-based authorization checks.
- Hardened `UseCentralDB` to prevent overwriting the current database connection if the central database is not available in the context.

## [v0.0.46] - 2026-04-21
### Fixed
- **Critical:** Fixed a session race condition by ensuring the session store and the main application share the same database connection pool. This guarantees immediate visibility of session updates (like login) across subsequent requests.
- Further refined cookie domain normalization to improve cross-browser compatibility.

## [v0.0.45] - 2026-04-21
### Fixed
- Critical session reliability in production:
    - Added thread-safe `sync.RWMutex` to `sessionData` to prevent data races during database updates.
    - Normalized cookie domain handling (explicitly avoiding leading dots for `localhost`).
    - Forced more aggressive cookie synchronization by always setting the cookie in the `load` phase.
    - Improved data consistency by safely cloning session state under lock before database persistence.

## [v0.0.44] - 2026-04-21
### Added
- Comprehensive `slog` logging for session load, save, and database persistence failures. This provides better diagnostics for production environments.

## [v0.0.43] - 2026-04-21
### Fixed
- Resolved circular dependency between `tracks` and `tracks/session` by moving `IsSecure` to the `session` package.
- Restored missing `net` import in `router.go`.

## [v0.0.42] - 2026-04-21
### Added
- `Config.Secure` flag to explicitly enable HTTPS across the application.
- Improved HTTPS detection: `tracks.IsSecure(r)` now respects the `X-Forwarded-Proto` header, fixing session issues behind reverse proxies.
### Changed
- Refactored `DomainMiddleware` and `DomainFromContext` into `tracks` core for better consistency.

## [v0.0.41] - 2026-04-21
### Fixed
- Session management on subdomains: ensured global middlewares are correctly copied during router cloning.
- Session persistence reliability: implemented a `ResponseWriter` wrapper to save sessions before headers or data are sent, preventing race conditions.
- Session state synchronization: added logic to issue new session cookies if a session ID changes (e.g., during invalidation/login).
### Security
- Replaced insecure `randomString` implementation with a cryptographically secure one using `crypto/rand`.
### Performance
- Optimized session middleware to avoid redundant execution if a session is already present in the request context.

## [v0.0.26] - 2026-02-17
### Added
- Plug-and-play mail driver registry.
- Module-specific configuration support in the `tracks` core.
- `Config()` method on the `Router` interface.
- Automatic driver registration for `log` and `smtp` mail drivers.

## [v0.0.25] - 2026-02-16
### Added
- Post-user-creation hooks in the authentication module.

## [v0.0.24] - 2026-02-16
### Added
- New `mail` module with SMTP and Log drivers.

## [v0.0.23] - 2026-02-15
### Added
- Internationalization (i18n) support.
### Fixed
- Resource naming in controllers.

## [v0.0.22] - 2026-02-08
### Changed
- Updated OpenTelemetry instrumentation.
### Fixed
- Import path in the disk storage driver.

## [v0.0.21] - 2026-02-06
### Added
- New `storage` module for unified file handling.
- Disk storage driver (built-in).
- S3 storage driver (as a separate module `storage/s3`).
- GCS storage driver (as a separate module `storage/gcs`).
- Database-backed blob tracking with `tracks_blobs` table.
- Multi-tenant isolation for storage keys.

## [v0.0.19] - 2026-01-31
### Added
- Lifecycle hooks support in Repository (`BeforeCreate`, `AfterCreate`, `BeforeUpdate`, `AfterUpdate`, `BeforeDelete`, `AfterDelete`).
- `database.SkipHooks` to bypass lifecycle hooks when needed.

## [v0.0.18] - 2026-01-31
### Added
- Atomic update support in Repository via `AtomicUpdate` method.

## [v0.0.17] - 2026-01-31
### Added
- Database transaction support via `database.WithTransaction`.
- Nested transaction support (via reuse).

### Fixed
- Compilation error in `database/sqlite/driver.go` regarding `otelsql.RegisterDBStatsMetrics` return values.
