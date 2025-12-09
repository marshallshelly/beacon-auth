---
title: Gin Integration
description: Complete guide to integrating BeaconAuth with Gin
---

BeaconAuth provides native integration with [Gin](https://github.com/gin-gonic/gin).

## Installation

```bash
go get github.com/marshallshelly/beacon-auth
go get github.com/gin-gonic/gin
```

## Basic Setup

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/marshallshelly/beacon-auth/adapters/memory"
    "github.com/marshallshelly/beacon-auth/auth"
    beacongin "github.com/marshallshelly/beacon-auth/integrations/gin"
    "github.com/marshallshelly/beacon-auth/session"
)

func main() {
    r := gin.New()

    // Setup...
    dbAdapter := memory.New()
    sessionManager, _ := session.NewManager(&session.Config{
        CookieName: "session",
        Secret:     "secret-key-at-least-32-bytes-long",
    }, dbAdapter)

    // Middleware
    r.Use(beacongin.SessionMiddleware(sessionManager))

    // Auth Routes
    authHandler := beacongin.NewHandler(dbAdapter, sessionManager, &auth.Config{})
    authHandler.RegisterRoutes(r)

    // Protected Routes
    authorized := r.Group("/api")
    authorized.Use(beacongin.RequireAuthJSON(sessionManager))
    {
        authorized.GET("/profile", func(c *gin.Context) {
            user := beacongin.GetUser(c)
            c.JSON(200, gin.H{"user": user})
        })
    }

    r.Run(":8080")
}
```

## Middleware

- `SessionMiddleware`: Sets `session` and `user` keys in Gin context and updates request context.
- `RequireAuth`: Redirects unauthenticated users.
- `RequireAuthJSON`: Returns JSON error for API.

## Helpers

- `GetUser(c)`: Returns `*core.User`.
- `GetSession(c)`: Returns `*core.Session`.
- `GetTenant(c)`: Returns tenant string.
