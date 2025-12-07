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

- [x] Core infrastructure
- [x] Configuration system
- [x] Context management
- [x] Error handling
- [x] Adapter system (Memory, PostgreSQL)
- [x] Session management (Multi-layer: Cookie, Redis, Database)
- [x] Password hashing (Argon2id)
- [x] Secure token generation
- [x] Email/Password authentication handlers
- [x] Framework integrations (Fiber with multi-tenant support)
- [ ] OAuth providers
- [ ] Additional plugins (2FA, magic link, etc.)
- [ ] Additional framework integrations (Chi, Gin, Echo)
- [ ] Additional database adapters (MySQL, MongoDB, SQLite)
- [ ] Documentation website

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
