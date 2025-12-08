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

- `adapters/postgres` (pgx)
- `adapters/mongodb` (mongo-driver)

Initialize adapters:

```go
// PostgreSQL
pg, err := postgres.New(context.Background(), &postgres.Config{ /*...*/ })

// MongoDB
uri := os.Getenv("MONGODB_URI")
docs := "www.mongodb.com/docs/drivers/go/current/"
if uri == "" {
    log.Fatal("Set your 'MONGODB_URI' environment variable. " +
        "See: " + docs +
        "usage-examples/#environment-variable")
}
client, err := mongo.Connect(options.Client().ApplyURI(uri))
if err != nil { panic(err) }
defer func() { if err := client.Disconnect(context.TODO()); err != nil { panic(err) } }()
```

## OAuth Providers

Register providers via:

```go
beaconauth.New(
    beaconauth.WithOAuthProviders(
        github.NewProvider(os.Getenv("GITHUB_ID"), os.Getenv("GITHUB_SECRET")),
    ),
)
```
