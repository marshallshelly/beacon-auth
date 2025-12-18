# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.6.3] - 2025-12-18

### Fixed

- **Session Creation in Multi-Tenant Environments**: Fixed "Failed to create session" error when session manager performed redundant user lookup that could fail in multi-tenant contexts.
  - Added `User` field to `SessionOptions` to allow passing pre-fetched user data
  - Updated `session.Manager.Create()` to use pre-fetched user from options, avoiding redundant database lookup
  - Updated `auth.Handler` SignIn and SignUp to pass user in `SessionOptions`
  - This fix eliminates the redundant user lookup that was causing failures when adapters were scoped to different tenant databases

## [0.6.2] - 2025-12-18

### Fixed

- **Critical: SignIn Credential Retrieval**: Fixed "Failed to retrieve credentials" error during email/password authentication.
  - `auth/handlers.go` was using incorrect column names (`provider`, `password_hash`) instead of the correct better-auth schema columns (`provider_id`, `provider_type`, `password`).
  - Updated `createUserWithPassword()` to use `InternalAdapter.CreateCredentialAccount()` which uses correct column names.
  - Updated `getUserPasswordHash()` to query for `provider_type="credential"` and read from `password` column.
  - **Migration Required**: Users upgrading from v0.6.1 or earlier need to migrate their database schema. See `docs/concepts/database.md` for migration guide.

### Changed

- **Documentation**: Updated `docs/concepts/database.md` to include `provider_type` column in accounts table schema and added migration guide for users on older schemas.

## [0.6.1] - 2025-12-09

### Fixed

- **Adapter Stability**: Fixed unchecked error returns (`rows.Close()`) in MySQL, MSSQL, and SQLite adapters to prevent potential resource leaks.
- **CLI Cleanliness**: Fixed static analysis warnings in the schema generator and CLI entry point.
- **Configuration**: Updated `InternalAdapter` instantiation to correctly pass configuration map (fixes `nil` pointer issues in some contexts).

## [0.6.0] - 2025-12-09

### Added

- **BeaconAuth CLI**: New command-line tool (`cmd/beacon`) for easy setup and management.
  - `beacon generate`: Automatically generates SQL schemas for your database and enabled plugins.
  - Supports `postgres`, `mysql`, `sqlite`, and `mssql` adapters.
- **Flexible ID Strategies**: Core adapters now support multiple ID generation strategies:
  - `string`: Application-side random string IDs (default, CUID-like).
  - `uuid`: Database-side UUID generation (e.g., `gen_random_uuid()`).
  - `serial`: Database-side auto-incrementing integers.
- **Documentation**:
  - New **CLI Tool** guide in Concepts.
  - Split Plugin documentation into dedicated pages (`email-password`, `twofa`, `oauth`).
  - Updated **Quickstart** guide to recommend CLI-based schema generation.
  - Updated website sidebar navigation.

### Changed

- **Internal Adapter**: Refactored `InternalAdapter` to accept configuration for `IDStrategy`, enabling native DB IDs.

## [0.5.0] - 2025-12-09

### Added

- **Database Adapters**: Added production-ready adapters for:

  - **MySQL**: Native `database/sql` implementation.
  - **SQLite**: Pure Go implementation using `modernc.org/sqlite`.
  - **MSSQL**: Native SQL Server implementation using `microsoft/go-mssqldb` with support for `OUTPUT` and pagination.

## [0.4.0] - 2025-12-09

### Added

- **Framework Integrations**: Added comprehensive integrations for popular Go web frameworks:

  - **Standard net/http**: Native `http.Handler` support compatible with standard mux.
  - **Chi**: Native middleware and route registration for `go-chi/chi`.
  - **Gin**: Native middleware and helpers for `gin-gonic/gin`.
  - **Echo**: Native middleware and utilities for `labstack/echo`.
  - **Gorilla Mux**: Native support for `gorilla/mux` router.
  - All integrations include support for:
    - Session management middleware
    - Authentication helpers (`GetUser`, `GetSession`)
    - Multi-tenant path/header extraction
    - User/Session keys in context

- **Documentation**: Added dedicated documentation guides for all new integrations.

## [0.3.0] - 2025-12-09

### Added

- **Roles & Permissions**: Added `Role` field to User struct for RBAC support.
- **User Management**: Added `Banned`, `BanReason`, and `BanExpires` fields to User struct.
- **Additional Fields**: Exposed `Metadata` field in User, Session, and Account structs to support arbitrary fields from database columns.
- **Session**: Added `ImpersonatedBy` field for admin impersonation checks.

### Changed

- **Schema Update**: Renamed `Account` struct fields to match better-auth schema (`Provider` -> `ProviderID`, added `AccessTokenExpiresAt`).
- **Schema Update**: Renamed `Verification` struct fields (`Token` -> `Value`, removed `Type`).

## [0.2.0] - 2025-12-08

### Added

- **Apple OAuth Provider**: Complete Sign in with Apple implementation
  - JWT client secret generation using ES256
  - ID token based authentication (Apple doesn't have userinfo endpoint)
  - Private email relay support
  - Refresh token support
  - Public key verification (JWKS)
  - Team ID, Key ID, and Private Key configuration

### Documentation

- **Fiber Integration Guide**: Comprehensive documentation for Fiber framework
  - Session middleware and authentication helpers
  - Multi-tenant architecture with subdomain extraction
  - Tenant-specific database isolation
  - Complete examples for single and multi-tenant apps
  - Custom middleware patterns (role-based access, email verification)
- Updated all provider documentation with Apple examples
- Added Apple configuration to quickstart guide
- Updated OAuth provider reference documentation
- All provider count updated from 3 to 4 providers
- README updated with Fiber-first quick start example

## [0.1.3] - 2025-12-08

### Added

- **OAuth Providers**: Google and Discord providers with full feature set
  - Google: PKCE support, ID token handling, refresh tokens
  - Discord: Custom/default avatar handling, animated avatar detection
- **Two-Factor Authentication**: Complete TOTP implementation
  - Backup codes generation (10 codes per user)
  - Secure one-time backup code consumption
  - Login interception flow
  - Complete database integration
- **MongoDB Adapter**: Production-ready MongoDB adapter
  - BSON query building
  - Aggregation support
  - Transaction support
- **CI/CD Improvements**:
  - Redis service for session tests
  - Optimized test exclusions for faster builds
  - Database-independent release workflow

### Changed

- Updated User model with `TwoFactorEnabled` field
- Enhanced OAuth token handling with expiration tracking
- Improved error messages (lowercase per Go style guide)

### Documentation

- Added comprehensive OAuth provider examples (GitHub, Google, Discord)
- Updated all documentation with 2FA backup codes
- Added OAuth Providers section to README
- Updated quickstart guide with complete examples
- Enhanced plugins and configuration reference docs
- Added database schema for backup codes table

### Fixed

- CI workflow now properly handles Redis tests
- Release workflow skips database adapters for faster builds
- GoReleaser config excludes database tests

## [0.1.2] - 2025-12-08

### Added

- Core authentication infrastructure
- Multi-layer session management (Cookie, Redis, Database)
- Argon2id password hashing
- PostgreSQL and in-memory database adapters
- Fiber framework integration with multi-tenant support
- Plugin system foundation
- HTTP authentication handlers (SignUp, SignIn, SignOut, GetSession)
- Session and authentication middleware
- Initial OAuth plugin with GitHub provider
- Email/Password authentication plugin
- Two-Factor Authentication plugin (initial)

### Changed

- Module renamed from `beaconauth` to `beacon-auth` for consistency

### Documentation

- Comprehensive CLAUDE.md guide for development
- README with quick start guide
- Implementation plan document

## [0.1.0] - Initial Release

### Initial Development Phase

- Project structure and architecture
- Core types and interfaces
- Configuration system with functional options
- Context management
- Error handling system
- Logging infrastructure

[Unreleased]: https://github.com/marshallshelly/beacon-auth/compare/v0.6.3...HEAD
[0.6.3]: https://github.com/marshallshelly/beacon-auth/compare/v0.6.2...v0.6.3
[0.6.2]: https://github.com/marshallshelly/beacon-auth/compare/v0.6.1...v0.6.2
[0.6.1]: https://github.com/marshallshelly/beacon-auth/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/marshallshelly/beacon-auth/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/marshallshelly/beacon-auth/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/marshallshelly/beacon-auth/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/marshallshelly/beacon-auth/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/marshallshelly/beacon-auth/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/marshallshelly/beacon-auth/compare/v0.1.3...v0.2.0
[0.1.3]: https://github.com/marshallshelly/beacon-auth/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/marshallshelly/beacon-auth/compare/v0.1.0...v0.1.2
[0.1.0]: https://github.com/marshallshelly/beacon-auth/releases/tag/v0.1.0
