# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Core authentication infrastructure
- Multi-layer session management (Cookie, Redis, Database)
- Argon2id password hashing
- PostgreSQL and in-memory database adapters
- Fiber framework integration with multi-tenant support
- Plugin system foundation
- HTTP authentication handlers (SignUp, SignIn, SignOut, GetSession)
- Session and authentication middleware

### Changed

- Module renamed from `beaconauth` to `beacon-auth` for consistency

### Documentation

- Comprehensive CLAUDE.md guide for development
- README with quick start guide
- Implementation plan document

## [0.1.0] - Unreleased

### Initial Development Phase

- Project structure and architecture
- Core types and interfaces
- Configuration system with functional options
- Context management
- Error handling system
- Logging infrastructure
