---
title: Echo Integration
description: Complete guide to integrating BeaconAuth with Echo
---

BeaconAuth provides native integration with [Echo](https://echo.labstack.com/).

## Installation

```bash
go get github.com/marshallshelly/beacon-auth
go get github.com/labstack/echo/v4
```

## Basic Setup

```go
package main

import (
    "github.com/labstack/echo/v4"
    "github.com/marshallshelly/beacon-auth/adapters/memory"
    "github.com/marshallshelly/beacon-auth/auth"
    beaconecho "github.com/marshallshelly/beacon-auth/integrations/echo"
    "github.com/marshallshelly/beacon-auth/session"
)

func main() {
    e := echo.New()

    // Setup...
    dbAdapter := memory.New()
    sessionManager, _ := session.NewManager(&session.Config{
        CookieName: "session",
        Secret:     "secret-key-at-least-32-bytes-long",
    }, dbAdapter)

    // Middleware
    e.Use(beaconecho.SessionMiddleware(sessionManager))

    // Auth Routes
    authHandler := beaconecho.NewHandler(dbAdapter, sessionManager, &auth.Config{})
    authHandler.RegisterRoutes(e.Group(""))

    // Protected Routes
    api := e.Group("/api")
    api.Use(beaconecho.RequireAuthJSON(sessionManager))
    api.GET("/profile", func(c echo.Context) error {
        user := beaconecho.GetUser(c)
        return c.JSON(200, user)
    })

    e.Start(":8080")
}
```

## Middleware

- `SessionMiddleware`: Sets `session` and `user` keys in context.
- `RequireAuth/RequireAuthJSON`
- `TenantMiddleware`

## Helpers

- `GetUser(c)`
- `GetSession(c)`
- `GetTenant(c)`
