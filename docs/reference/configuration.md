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
        ExpiresIn:      30 * 24 * time.Hour,
        CookieSecure:   true,  // Set to true in production
        CookieHTTPOnly: true,
        CookieSameSite: "Lax",
    }),
)
```

### Session Config Reference

- `CookieName`: Name of the session cookie.
- `ExpiresIn`: Duration before session expires.
- `CookieSecure`: Ensure cookie is only sent over HTTPS.
- `CookieHTTPOnly`: Prevent JS access to cookie (XSS protection).
- `CookieSameSite`: CSRF protection ("Lax", "Strict", "None").
