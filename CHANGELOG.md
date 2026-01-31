# Changelog

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
