# Changelog

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
