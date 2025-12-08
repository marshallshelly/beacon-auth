# BeaconAuth

A modular, plugin-based authentication library for Go with support for multiple databases, OAuth providers, and web frameworks.

## Features

- 🔌 **Plugin Architecture** - Extend with OAuth, 2FA, magic links, and custom plugins
- 🗄️ **Database Agnostic** - PostgreSQL, MySQL, MongoDB, SQLite, or custom adapters
- ⚡ **Framework Flexible** - Fiber, Chi, Gin, Echo, Gorilla Mux, or standard net/http
- 🔒 **Secure by Default** - CSRF protection, rate limiting, secure sessions
- 📦 **Production Ready** - Built-in session management, migrations, and security features

## Installation

```bash
go get github.com/marshallshelly/beacon-auth
```

## Quick Start

```go
package main

import (
    "log"
    "net/http"

    "github.com/marshallshelly/beacon-auth"
    "github.com/marshallshelly/beacon-auth/adapters/postgres"
)

func main() {
    // Initialize database adapter
    adapter, err := postgres.New(&postgres.Config{
        Host:     "localhost",
        Port:     5432,
        User:     "postgres",
        Password: "password",
        Database: "myapp",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Initialize BeaconAuth
    auth, err := beaconauth.New(
        beaconauth.WithAdapter(adapter),
        beaconauth.WithSecret("your-secret-key"),
        beaconauth.WithBaseURL("http://localhost:3000"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer auth.Close()

    // Mount auth routes
    http.Handle("/auth/", http.StripPrefix("/auth", auth.Handler()))

    // Protected route
    http.Handle("/api/", auth.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        session := beaconauth.GetSession(r.Context())
        // Use session...
    })))

    log.Fatal(http.ListenAndServe(":3000", nil))
}
```

## Project Status

🚧 **In Development** - BeaconAuth is currently under active development. The API may change.

### Current Progress

**Core Infrastructure:**

- [x] Core infrastructure
- [x] Configuration system
- [x] Context management
- [x] Error handling
- [x] Logging system

**Data Layer:**

- [x] Adapter system with automatic type transformations
- [x] Memory adapter (for testing)
- [x] PostgreSQL adapter
- [x] MongoDB adapter
- [ ] MySQL adapter
- [ ] SQLite adapter

**Session Management:**

- [x] Multi-layer session management (Cookie, Redis, Database)
- [x] Session strategies (Redis-first, DB-first, Cookie-only, etc.)
- [x] Automatic TTL and cleanup
- [x] Session middleware

**Authentication:**

- [x] Password hashing (Argon2id)
- [x] Secure token generation
- [x] Email/Password authentication handlers

**Plugins:**

- [x] Plugin system foundation
- [x] OAuth plugin (GitHub provider implemented)
- [x] Email/Password plugin
- [x] Two-Factor Authentication (TOTP + backup codes)
- [ ] Magic link plugin
- [ ] Passkey/WebAuthn plugin
- [ ] Additional OAuth providers (Google, Discord, Apple, etc.)

**Framework Integrations:**

- [x] Fiber (with multi-tenant support)
- [ ] Chi
- [ ] Gin
- [ ] Echo
- [ ] Standard net/http

**Documentation:**

- [x] Comprehensive inline documentation
- [x] Example code and tests
- [ ] Documentation website (in progress)

## Architecture

BeaconAuth follows a modular architecture with these core components:

- **Core** - Main authentication logic and interfaces
- **Adapters** - Database adapters for different databases
- **Plugins** - Extensible plugin system for adding features
- **Integrations** - Framework-specific integrations
- **Session** - Multi-layer session management
- **Security** - Rate limiting, CSRF protection, password hashing

## Documentation

Coming soon! Full documentation will be available at [beaconauth.dev](https://beaconauth.dev)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details

## Acknowledgments

Inspired by [better-auth](https://github.com/better-auth/better-auth) - bringing modern auth patterns to Go.
