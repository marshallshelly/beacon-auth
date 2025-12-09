---
title: Chi Integration
description: Complete guide to integrating BeaconAuth with go-chi
---

BeaconAuth provides native integration with [Chi](https://github.com/go-chi/chi).

## Installation

```bash
go get github.com/marshallshelly/beacon-auth
go get github.com/go-chi/chi/v5
```

## Basic Setup

```go
package main

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/marshallshelly/beacon-auth/adapters/memory"
    "github.com/marshallshelly/beacon-auth/auth"
    beaconchi "github.com/marshallshelly/beacon-auth/integrations/chi"
    "github.com/marshallshelly/beacon-auth/session"
)

func main() {
    r := chi.NewRouter()

    // Setup...
    dbAdapter := memory.New()
    sessionManager, _ := session.NewManager(&session.Config{
        CookieName: "session",
        Secret:     "secret-key-at-least-32-bytes-long",
    }, dbAdapter)

    // Middleware
    r.Use(beaconchi.SessionMiddleware(sessionManager))

    // Auth Routes
    authHandler := beaconchi.NewHandler(dbAdapter, sessionManager, &auth.Config{})
    authHandler.RegisterRoutes(r)

    // Protected Routes
    r.Group(func(r chi.Router) {
        r.Use(beaconchi.RequireAuth(sessionManager))
        r.Get("/profile", func(w http.ResponseWriter, r *http.Request) {
            // ...
        })
    })

    http.ListenAndServe(":3000", r)
}
```

## Middleware

- `SessionMiddleware`: Loads session into context.
- `RequireAuth`: Redirects unauthenticated users.
- `TenantMiddleware`: Multi-tenant support compatible with Chi.

## Helpers

- `GetSession(w, r)`: HTTP handler to get current session JSON.
- `GetTenant(ctx)`: Retrieve tenant from context.
