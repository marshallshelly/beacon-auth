# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

[Unreleased]: https://github.com/marshallshelly/beacon-auth/compare/v0.1.3...HEAD
[0.1.3]: https://github.com/marshallshelly/beacon-auth/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/marshallshelly/beacon-auth/compare/v0.1.0...v0.1.2
[0.1.0]: https://github.com/marshallshelly/beacon-auth/releases/tag/v0.1.0
