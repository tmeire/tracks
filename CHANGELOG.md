# Changelog

## [v0.0.81] - 2026-11-13
### Security
- Sessions: Sessions expire server-side — 30 days after creation and after 7 days without a request — and `Authenticate` issues a new session ID on login. `sessions.absolute_timeout` and `sessions.idle_timeout` (Go duration strings; `"0"` disables a limit) override the defaults. `Store.DeleteExpired` removes expired rows from the database store; nothing schedules it. Existing sessions older than the limits are logged out.
- Authentication: Activation tokens expire after 7 days (`authentication.ActivationTokenTTL`); the expiry is part of the token, so no schema change is needed. Tokens issued before this release are treated as expired and need a new invite. `User.ActivationTokenExpiry` and `User.ActivationTokenValid` expose the check.
- Authentication: `SetPasswordWithToken` rejects an empty token. It matched every user whose stored token is empty and set that user's password.
- CSRF: The CSRF cookie is `Secure` behind a TLS-terminating proxy (it followed `r.TLS` only). `CSRFConfig.Secure` forces the flag; `router.CSRFProtection` sets it from `router.Secure()`.
### Fixed
- SQLite: Every connection gets `busy_timeout` (5000 ms, `busy_timeout_ms` in the sqlite config; negative disables), including tenant databases and the migration runner. `foreign_keys: true` in the sqlite config turns on foreign key enforcement; it is off by default because table-rebuild migrations would cascade-delete child rows.
- Actions: HTML error pages are sent with their status code; they were rendered with 200 OK.
- Logging: Route registration, the tenant splitter and the feature flag middleware log through `slog.Debug` instead of printing to stdout.

## [v0.0.80] - 2026-11-07
### Fixed
- MCP: Served the authorization server metadata at `/.well-known/oauth-authorization-server` (the RFC 8414 location for an origin issuer) in addition to the path-scoped variant; clients that read it from the issuer root saw a 404 and rejected dynamic client registration.

## [v0.0.79] - 2026-11-07
### Fixed
- MCP: Didn't append the internal config port to the issuer origin on secure servers — behind a TLS terminator the config port is the proxy's internal port, so advertised OAuth endpoints pointed at a closed port and clients timed out. Hosts behind a proxy set `ServerInfo.Issuer` explicitly.

## [v0.0.78] - 2026-11-06
### Added
- MCP: OAuth 2.1 authorization for `modules/mcp` — dynamic client registration (RFC 7591), authorization code + PKCE with S256 (RFC 7636), token endpoint with refresh rotation (RFC 6749), and RFC 8414/9728 well-known metadata. `ServerInfo.Issuer` and `ServerInfo.OAuthBase` select the public origin and OAuth endpoint prefix; `WithOAuth(consentSubject)` enables the flow and `Server.BearerOrSession` accepts an access token or the host's session on the JSON-RPC route (a login redirect becomes 401 + WWW-Authenticate so clients can start the flow). Grants and tokens are kept in memory: a restart logs clients out and they re-consent. The host provides the consent view at `views/mcp/oauth_consent.gohtml`.

## [v0.0.77] - 2026-10-27
### Added
- MCP: New `modules/mcp` package — a domain-agnostic Model Context Protocol engine for Tracks applications. It provides JSON-RPC 2.0 message handling, a tool registry with first-class destructive-tool filtering (`Tool.Destructive` + `WithReadOnly`), a stdio transport, and a stateless Streamable HTTP transport (`Register` mounts GET discovery and POST JSON-RPC on any route, and refuses to mount without authentication middleware). Supports protocol versions 2024-11-05 through 2025-06-18.
### Changed
- Templates: Dynamic template resolution is cached when `development` is false; template edits are picked up per-request only in development mode. `newTemplates` takes the development flag.

## [v0.0.76] - 2026-10-10
### Fixed
- Templates: Partials (`_name.gohtml`) are found when the views directory is a symlink. `filepath.Walk` doesn't descend into a symlinked root, so a layout using `{{ template "name" . }}` failed with "no such template" when the views were linked in, e.g. next to a package's tests.

## [v0.0.75] - 2026-10-06
### Fixed
- Redirects: HTML responses now honor an explicit 3xx `Response.StatusCode` (e.g. 301 Moved Permanently) instead of always sending 303 See Other. 303 stays the default when no redirect status is given, which suits redirects after form posts. HTMX requests still get `HX-Redirect` with 202.
- Redirects: Removed the `DEBUG: renderHTML redirecting` line that was printed to stdout on every redirect.

## [v0.0.74] - 2026-10-06
### Added
- i18n: Opt-in `i18n` config block (`default`, `locales`, `strategy`, `base_url`). With `"strategy": "path"`, `/nl/...` serves Dutch from the same routes (the prefix is stripped before routing), `/en/...` redirects to the unprefixed URL, and cookies or headers never change the language of a URL.
- i18n: Template helpers `localize`, `alternates`, `alternate_links` and `canonical_url`; `suggested_locale` view var.
- i18n: `tracks.SetAlternates`, `tracks.LocalizePath`, `tracks.T`, `i18n.URLFor`, `i18n.CanonicalURL` and `i18n.Alternates` helpers.
- Testing: `TestConfig.I18n` to test localized routes.
### Changed
- i18n: Root-relative redirects (`Response.Location`, `tracks.Redirect`, `Router.Redirect`) keep the language prefix of the current page in path mode.
- i18n: Missing translation keys are logged once per locale and key.
### Fixed
- i18n: The legacy `canonical_*` view vars were hardcoded to `https://floralynx.com`; they're now derived from `base_domain` and `secure`.

## [v0.0.60] - 2026-05-14
### Fixed
- BaseController: Added a nil check for the router in `Scheme()` to prevent panics when the controller is not fully initialized.

## [v0.0.58] - 2026-05-13
### Added
- Router: Added router injection in `Controller` and `ControllerAtPath` to allow controllers to access the router instance they are registered on.

## [v0.0.57] - 2026-05-12
### Changed
- Templates: Improved partial naming logic to be more robust.

## [v0.0.56] - 2026-05-12
### Fixed
- Templates: Fixed dummy function signatures and partial loading logic.

## [v0.0.55] - 2026-05-12
### Changed
- Templates: Updated design system components in built-in templates.

## [v0.0.54] - 2026-05-12
### Added
- Auth: Added `HasPassword` method to the `User` model.

## [v0.0.53] - 2026-05-12
### Added
- Templates: Added `icon` template helper.
### Changed
- OpenTelemetry: Cleaned up OTel stdout exporters.

## [v0.0.52] - 2026-05-09
### Fixed
- Authentication: Fixed user activation and password setup flow on subdomains by ensuring the central database is used instead of the tenant database.
### Changed
- Database: Moved `WithCentralDB` and `CentralDBFromContext` to the core `database` package to allow cross-module access without circular dependencies.
- Session: Added `WithContext` to allow setting a session in the context, primarily for testing purposes.

## [v0.0.51] - 2026-04-22
### Fixed
- Fixed compilation errors in `versionRouter` and `mockRouter` due to the missing `SkipDefaultMiddlewares()` method.

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
