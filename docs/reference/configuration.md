---
title: Configuration
description: Full reference for BeaconAuth configuration options.
---

BeaconAuth is highly configurable via the `beaconauth.New()` function options.

## Core Options

| Option                 | Description                                                      | Default |
| ---------------------- | ---------------------------------------------------------------- | ------- |
| `WithAdapter(adapter)` | **Required**. Database adapter instance.                         | `nil`   |
| `WithSecret(string)`   | **Required**. Secret key for signing tokens/cookies.             | `""`    |
| `WithBaseURL(string)`  | **Required**. Public URL of your app (e.g. `https://myapp.com`). | `""`    |
| `WithBasePath(string)` | URI path prefix for auth routes.                                 | `/auth` |

## Plugin Registration

Register plugins to enable authentication methods.

```go
beaconauth.New(
    // ...
    beaconauth.WithPlugins(
        emailpassword.New(),
        twofa.New(),
        oauth.New(github.Provider(...)),
    ),
)
```

## Session Configuration

Customize session behavior using `WithSessionConfig`:

```go
beaconauth.New(
    // ...
    beaconauth.WithSessionConfig(&core.SessionConfig{
        CookieName:     "myapp_session",
        CookieDomain:   "",
        CookiePath:     "/",
        CookieSecure:   true,  // Set to true in production
        CookieHTTPOnly: true,
        CookieSameSite: "lax",
        ExpiresIn:      30 * 24 * time.Hour,
        UpdateAge:      24 * time.Hour,
    }),
)
```

### Session Config Reference

- `CookieName`: Name of the session cookie.
- `CookieDomain`: Optional domain for the cookie (e.g. `.example.com`).
- `CookiePath`: Path for the cookie (default: `/`).
- `CookieSecure`: Ensure cookie is only sent over HTTPS.
- `CookieHTTPOnly`: Prevent JS access to cookie (XSS protection).
- `CookieSameSite`: CSRF protection ("lax", "strict", "none").
- `ExpiresIn`: Duration before session expires.
- `UpdateAge`: If session last-updated is older than this, refresh timestamp.

## Advanced Options

Use these to control security and logging:

```go
beaconauth.New(
    beaconauth.WithTrustedOrigins("https://app.example.com"),
    beaconauth.WithLogger(core.NewDefaultLogger()),
)
```

- `WithTrustedOrigins`: Configure allowed origins for CORS checks.
- `WithLogger`: Provide a custom logger implementation.

## Database Adapters

BeaconAuth supports pluggable adapters. Currently available:

- **Memory Adapter** (`adapters/memory`) - For testing and development
- **PostgreSQL** (`adapters/postgres`) - Production-ready with pgx driver
- **MongoDB** (`adapters/mongodb`) - Production-ready with mongo-driver

Initialize adapters:

```go
// PostgreSQL
import "github.com/marshallshelly/beacon-auth/adapters/postgres"

pg, err := postgres.New(context.Background(), &postgres.Config{
    Host:     "localhost",
    Port:     5432,
    Database: "auth",
    Username: "postgres",
    Password: "postgres",
})

// MongoDB
import "github.com/marshallshelly/beacon-auth/adapters/mongodb"

mongo, err := mongodb.New(context.Background(), &mongodb.Config{
    URI:      "mongodb://localhost:27017",
    Database: "auth",
})

// Memory (for testing)
import "github.com/marshallshelly/beacon-auth/adapters/memory"

mem := memory.New()
```

## OAuth Providers

BeaconAuth includes three OAuth providers out of the box.

### GitHub

```go
import "github.com/marshallshelly/beacon-auth/plugins/oauth/providers"

githubProvider := providers.NewGitHub(
    os.Getenv("GITHUB_CLIENT_ID"),
    os.Getenv("GITHUB_CLIENT_SECRET"),
    []string{"user:email"},
)
```

### Google (with PKCE)

```go
googleProvider := providers.NewGoogle(&providers.GoogleOptions{
    ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
    ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
    AccessType:   "offline", // Request refresh token
    Scopes:       []string{"openid", "email", "profile"}, // Optional, these are defaults
})
```

### Discord

```go
discordProvider := providers.NewDiscord(&providers.DiscordOptions{
    ClientID:     os.Getenv("DISCORD_CLIENT_ID"),
    ClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
    Prompt:       "none", // or "consent"
    Scopes:       []string{"identify", "email"}, // Optional, these are defaults
})
```

### Apple

```go
appleProvider := providers.NewApple(&providers.AppleOptions{
    ClientID:   os.Getenv("APPLE_SERVICE_ID"),
    TeamID:     os.Getenv("APPLE_TEAM_ID"),
    KeyID:      os.Getenv("APPLE_KEY_ID"),
    PrivateKey: os.Getenv("APPLE_PRIVATE_KEY"), // PEM format EC private key
    // Or use pre-generated client secret:
    // ClientSecret: os.Getenv("APPLE_CLIENT_SECRET"),
})
```

### Register with BeaconAuth

```go
import "github.com/marshallshelly/beacon-auth/plugins/oauth"

auth, _ := beaconauth.New(
    beaconauth.WithAdapter(adapter),
    beaconauth.WithPlugins(
        oauth.New(githubProvider, googleProvider, discordProvider, appleProvider),
    ),
)
```
