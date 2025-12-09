# BeaconAuth

[![Go Reference](https://pkg.go.dev/badge/github.com/marshallshelly/beacon-auth.svg)](https://pkg.go.dev/github.com/marshallshelly/beacon-auth)
[![Go Report Card](https://goreportcard.com/badge/github.com/marshallshelly/beacon-auth)](https://goreportcard.com/report/github.com/marshallshelly/beacon-auth)
[![CI](https://github.com/marshallshelly/beacon-auth/actions/workflows/ci.yml/badge.svg)](https://github.com/marshallshelly/beacon-auth/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/v/release/marshallshelly/beacon-auth.svg)](https://github.com/marshallshelly/beacon-auth/releases)

A modular, plugin-based authentication library for Go with support for multiple databases, OAuth providers, and web frameworks.

## Features

- 🔌 **Plugin Architecture** - Extend with OAuth, 2FA, magic links, and custom plugins
- 🗄️ **Database Agnostic** - PostgreSQL, MySQL, MongoDB, SQLite, or custom adapters
- ⚡ **Framework Flexible** - Fiber, Chi, Gin, Echo, Gorilla Mux, or standard net/http
- 🔒 **Secure by Default** - CSRF protection, rate limiting, secure sessions
- 🛠️ **CLI Tool** - Built-in CLI for easy schema generation and maintenance
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
    app := fiber.New()

    // Database adapter
    db, _ := postgres.New(context.Background(), &postgres.Config{
        Host: "localhost", Port: 5432, Database: "myapp",
        Username: "postgres", Password: "password",
    })
    defer db.Close()

    // Session manager
    sm, _ := session.NewManager(&session.Config{
        CookieName: "session", CookieSecure: true,
        ExpiresIn: 24 * time.Hour, EnableDBStore: true,
        Secret: "your-secret-at-least-32-bytes-long",
        Issuer: "myapp",
    }, db)

    // Session middleware
    app.Use(beaconfiber.SessionMiddleware(sm))

    // Auth routes
    authHandler := beaconfiber.NewHandler(db, sm, &auth.Config{
        MinPasswordLength: 8,
        AllowSignup:       true,
    })
    app.Post("/auth/signup", authHandler.SignUp)
    app.Post("/auth/signin", authHandler.SignIn)
    app.Post("/auth/signout", authHandler.SignOut)

    // Protected routes
    api := app.Group("/api")
    api.Use(beaconfiber.RequireAuthJSON(sm))
    api.Get("/profile", func(c *fiber.Ctx) error {
        user := beaconfiber.GetUser(c)
        return c.JSON(fiber.Map{"user": user})
    })

    log.Fatal(app.Listen(":3000"))
}
```

**Multi-Tenant Support:**

```go
// Tenant extraction from subdomain
tenantConfig := &beaconfiber.TenantConfig{
    BaseDomain: "myapp.com", // Extract from tenant1.myapp.com
}
app.Use(beaconfiber.TenantMiddleware(tenantConfig))

// Tenant-specific databases
app.Use(beaconfiber.TenantIsolationMiddleware(getTenantDB))

// Access tenant in routes
tenant := beaconfiber.GetTenant(c)
```

See [Fiber Integration Guide](./docs/integrations/fiber.md) for complete documentation.

## OAuth Providers

BeaconAuth supports multiple OAuth providers out of the box:

```go
import (
    "github.com/marshallshelly/beacon-auth/plugins/oauth"
    "github.com/marshallshelly/beacon-auth/plugins/oauth/providers"
)

// GitHub
githubProvider := providers.NewGitHub(
    "your-client-id",
    "your-client-secret",
    []string{"user:email"},
)

// Google (with PKCE)
googleProvider := providers.NewGoogle(&providers.GoogleOptions{
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret",
    AccessType:   "offline", // For refresh tokens
})

// Discord
discordProvider := providers.NewDiscord(&providers.DiscordOptions{
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret",
    Prompt:       "none", // or "consent"
})

// Add to BeaconAuth
auth, _ := beaconauth.New(
    beaconauth.WithAdapter(adapter),
    beaconauth.WithPlugins(
        oauth.New(githubProvider, googleProvider, discordProvider),
    ),
    // ... other options
)
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
- [x] MySQL adapter
- [x] SQLite adapter
- [x] MSSQL adapter

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
- [x] OAuth plugin (4 providers: GitHub, Google, Discord, Apple)
- [x] Email/Password plugin
- [x] Two-Factor Authentication (TOTP + backup codes)
- [ ] Magic link plugin
- [ ] Passkey/WebAuthn plugin
- [ ] Additional OAuth providers (Microsoft, Twitter, Facebook, etc.)

**Framework Integrations:**

- [x] Fiber (with multi-tenant support)
- [x] Chi
- [x] Gin
- [x] Echo
- [x] Standard net/http

**Documentation:**

- [x] Comprehensive inline documentation
- [x] Example code and tests
- [x] Documentation website (beaconauth.dev)

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
