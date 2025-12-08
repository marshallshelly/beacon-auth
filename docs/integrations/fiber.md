---
title: Fiber Integration
description: Complete guide to integrating BeaconAuth with Fiber web framework
---

BeaconAuth provides first-class integration with [Fiber v2](https://gofiber.io/), including support for multi-tenant architectures.

## Installation

The Fiber integration is included in the main BeaconAuth package:

```bash
go get github.com/marshallshelly/beacon-auth
go get github.com/gofiber/fiber/v2
```

## Basic Setup

### 1. Create Database Adapter and Session Manager

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/marshallshelly/beacon-auth/adapters/postgres"
    "github.com/marshallshelly/beacon-auth/auth"
    beaconfiber "github.com/marshallshelly/beacon-auth/integrations/fiber"
    "github.com/marshallshelly/beacon-auth/session"
)

func main() {
    app := fiber.New()

    // Create database adapter
    dbAdapter, err := postgres.New(context.Background(), &postgres.Config{
        Host:     "localhost",
        Port:     5432,
        Database: "myapp",
        Username: "postgres",
        Password: "password",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer dbAdapter.Close()

    // Create session manager
    sessionConfig := &session.Config{
        CookieName:     "app_session",
        CookieSecure:   true,   // Set to true in production
        CookieHTTPOnly: true,
        CookieSameSite: "lax",
        ExpiresIn:      7 * 24 * time.Hour,
        EnableDBStore:  true,
        Secret:         "your-secret-key-at-least-32-bytes",
        Issuer:         "myapp",
    }

    sessionManager, err := session.NewManager(sessionConfig, dbAdapter)
    if err != nil {
        log.Fatal(err)
    }

    // Continue with middleware and routes...
}
```

### 2. Add Session Middleware

The session middleware automatically loads sessions from cookies:

```go
// Add session middleware globally
app.Use(beaconfiber.SessionMiddleware(sessionManager))
```

### 3. Create Auth Handler and Routes

```go
// Create auth handler
authHandler := beaconfiber.NewHandler(dbAdapter, sessionManager, &auth.Config{
    MinPasswordLength:   8,
    RequireVerification: false,
    AllowSignup:         true,
})

// Auth routes
app.Post("/auth/signup", authHandler.SignUp)
app.Post("/auth/signin", authHandler.SignIn)
app.Post("/auth/signout", authHandler.SignOut)
app.Get("/auth/session", authHandler.GetSession)
```

### 4. Add Protected Routes

```go
// Protected routes require authentication
protected := app.Group("/api")
protected.Use(beaconfiber.RequireAuthJSON(sessionManager))

protected.Get("/profile", func(c *fiber.Ctx) error {
    user := beaconfiber.GetUser(c)
    return c.JSON(fiber.Map{
        "user": user,
    })
})
```

## Middleware

### Session Middleware

Automatically loads session and user data from cookies:

```go
app.Use(beaconfiber.SessionMiddleware(sessionManager))
```

**What it does:**

- Extracts session token from cookie
- Loads session and user from database
- Stores them in Fiber `Locals` for access in handlers

### RequireAuth

Redirects unauthenticated users (for HTML responses):

```go
// Redirects to /auth/signin if not authenticated
app.Use(beaconfiber.RequireAuth(sessionManager))
```

### RequireAuthJSON

Returns JSON error for unauthenticated requests (for API routes):

```go
api := app.Group("/api")
api.Use(beaconfiber.RequireAuthJSON(sessionManager))
```

**Response for unauthenticated:**

```json
{
  "error": "unauthorized",
  "message": "Authentication required"
}
```

## Helper Functions

### GetSession

Get the current session from Fiber context:

```go
session := beaconfiber.GetSession(c)
if session != nil {
    // Session is valid
}
```

### GetUser

Get the current user from Fiber context:

```go
user := beaconfiber.GetUser(c)
if user != nil {
    fmt.Println("User ID:", user.ID)
    fmt.Println("Email:", user.Email)
}
```

### GetUserID

Get just the user ID:

```go
userID := beaconfiber.GetUserID(c)
```

## Multi-Tenant Support

BeaconAuth includes built-in multi-tenant support for Fiber applications.

### Tenant Middleware

Extract tenant ID from subdomain or header:

```go
tenantConfig := &beaconfiber.TenantConfig{
    BaseDomain:    "myapp.com",
    TenantHeader:  "X-Tenant-ID",
    DefaultTenant: "",
}
app.Use(beaconfiber.TenantMiddleware(tenantConfig))
```

**Tenant extraction priority:**

1. `X-Tenant-ID` header (if TenantHeader is set)
2. Subdomain (e.g., `tenant1.myapp.com` → `tenant1`)
3. DefaultTenant (if provided)

### Tenant Isolation

Each tenant can have its own database:

```go
app.Use(beaconfiber.TenantIsolationMiddleware(func(tenantID string) (interface{}, error) {
    // Connect to tenant-specific database
    adapter, err := postgres.New(context.Background(), &postgres.Config{
        Host:     "localhost",
        Port:     5432,
        Database: "tenant_" + tenantID,
        Username: "postgres",
        Password: "password",
    })
    return adapter, err
}))
```

### Tenant Helpers

```go
// Get current tenant
tenant := beaconfiber.GetTenant(c)

// Get tenant-specific adapter
adapter := beaconfiber.GetAdapter(c)

// Require tenant (returns 400 if no tenant)
app.Use(beaconfiber.RequireTenant())
```

### Complete Multi-Tenant Example

```go
app := fiber.New()

// 1. Tenant extraction
app.Use(beaconfiber.TenantMiddleware(&beaconfiber.TenantConfig{
    BaseDomain: "soulcareuk.com",
}))

// 2. Load tenant-specific database
app.Use(beaconfiber.TenantIsolationMiddleware(getTenantAdapter))

// 3. Tenant-specific session manager
sessionManagerCache := make(map[string]*session.Manager)

app.Use(func(c *fiber.Ctx) error {
    tenant := beaconfiber.GetTenant(c)
    adapter := beaconfiber.GetAdapter(c).(core.Adapter)

    manager, exists := sessionManagerCache[tenant]
    if !exists {
        manager = createSessionManager(tenant, adapter)
        sessionManagerCache[tenant] = manager
    }

    c.Locals("sessionManager", manager)
    return c.Next()
})

// 4. Load sessions
app.Use(func(c *fiber.Ctx) error {
    manager := c.Locals("sessionManager").(*session.Manager)
    return beaconfiber.SessionMiddleware(manager)(c)
})

// 5. Protected routes
api := app.Group("/api")
api.Use(beaconfiber.RequireTenant())
api.Use(func(c *fiber.Ctx) error {
    manager := c.Locals("sessionManager").(*session.Manager)
    return beaconfiber.RequireAuthJSON(manager)(c)
})

api.Get("/profile", func(c *fiber.Ctx) error {
    user := beaconfiber.GetUser(c)
    tenant := beaconfiber.GetTenant(c)

    return c.JSON(fiber.Map{
        "tenant": tenant,
        "user":   user,
    })
})
```

## Custom Middleware Examples

### Require Email Verification

```go
func RequireVerified(c *fiber.Ctx) error {
    user := beaconfiber.GetUser(c)
    if user == nil || !user.EmailVerified {
        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "error":   "email_not_verified",
            "message": "Please verify your email address",
        })
    }
    return c.Next()
}

app.Use(RequireVerified)
```

### Role-Based Access Control

```go
func RequireRole(role string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        user := beaconfiber.GetUser(c)
        if user == nil {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
                "error": "unauthorized",
            })
        }

        // Check user role (implement based on your schema)
        userRole := getUserRole(user.ID)
        if userRole != role {
            return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
                "error": "insufficient_permissions",
            })
        }

        return c.Next()
    }
}

admin := app.Group("/admin")
admin.Use(RequireRole("admin"))
```

## Complete Example

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/cors"
    "github.com/gofiber/fiber/v2/middleware/logger"
    "github.com/marshallshelly/beacon-auth/adapters/postgres"
    "github.com/marshallshelly/beacon-auth/auth"
    beaconfiber "github.com/marshallshelly/beacon-auth/integrations/fiber"
    "github.com/marshallshelly/beacon-auth/session"
)

func main() {
    app := fiber.New()

    // Middleware
    app.Use(cors.New())
    app.Use(logger.New())

    // Database adapter
    db, _ := postgres.New(context.Background(), &postgres.Config{
        Host:     "localhost",
        Port:     5432,
        Database: "myapp",
        Username: "postgres",
        Password: "password",
    })
    defer db.Close()

    // Session manager
    sm, _ := session.NewManager(&session.Config{
        CookieName:     "session",
        CookieSecure:   true,
        CookieHTTPOnly: true,
        ExpiresIn:      24 * time.Hour,
        EnableDBStore:  true,
        Secret:         "your-secret-at-least-32-bytes-long",
        Issuer:         "myapp",
    }, db)

    // Session middleware
    app.Use(beaconfiber.SessionMiddleware(sm))

    // Auth handler
    authHandler := beaconfiber.NewHandler(db, sm, &auth.Config{
        MinPasswordLength: 8,
        AllowSignup:       true,
    })

    // Public routes
    app.Post("/auth/signup", authHandler.SignUp)
    app.Post("/auth/signin", authHandler.SignIn)
    app.Post("/auth/signout", authHandler.SignOut)

    // Protected routes
    api := app.Group("/api")
    api.Use(beaconfiber.RequireAuthJSON(sm))

    api.Get("/profile", func(c *fiber.Ctx) error {
        return c.JSON(beaconfiber.GetUser(c))
    })

    log.Fatal(app.Listen(":3000"))
}
```

## API Reference

### Types

```go
type TenantConfig struct {
    BaseDomain    string // Base domain for subdomain extraction
    TenantHeader  string // Header to check for tenant ID
    DefaultTenant string // Fallback tenant
    TenantKey     string // Locals key for storing tenant
}
```

### Functions

- `SessionMiddleware(manager *session.Manager) fiber.Handler`
- `RequireAuth(manager *session.Manager) fiber.Handler`
- `RequireAuthJSON(manager *session.Manager) fiber.Handler`
- `GetSession(c *fiber.Ctx) *core.Session`
- `GetUser(c *fiber.Ctx) *core.User`
- `GetUserID(c *fiber.Ctx) string`
- `TenantMiddleware(config *TenantConfig) fiber.Handler`
- `RequireTenant() fiber.Handler`
- `GetTenant(c *fiber.Ctx) string`
- `GetAdapter(c *fiber.Ctx) interface{}`
- `TenantIsolationMiddleware(getTenantAdapter func(tenantID string) (interface{}, error)) fiber.Handler`

## See Also

- [Session Management](/docs/guides/sessions)
- [Multi-Tenant Architecture](/docs/guides/multi-tenant)
- [Authentication](/docs/guides/authentication)
