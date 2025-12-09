---
title: Gorilla Mux Integration
description: Complete guide to integrating BeaconAuth with Gorilla Mux
---

BeaconAuth provides native integration with [Gorilla Mux](https://github.com/gorilla/mux).

## Installation

```bash
go get github.com/marshallshelly/beacon-auth
go get github.com/gorilla/mux
```

## Basic Setup

```go
package main

import (
    "net/http"
    "github.com/gorilla/mux"
    "github.com/marshallshelly/beacon-auth/adapters/memory"
    "github.com/marshallshelly/beacon-auth/auth"
    beaconmux "github.com/marshallshelly/beacon-auth/integrations/mux"
    "github.com/marshallshelly/beacon-auth/session"
)

func main() {
    r := mux.NewRouter()

    // Setup...
    dbAdapter := memory.New()
    sessionManager, _ := session.NewManager(&session.Config{
        CookieName: "session",
        Secret:     "secret-key-at-least-32-bytes-long",
    }, dbAdapter)

    // Middleware
    r.Use(beaconmux.SessionMiddleware(sessionManager))

    // Auth Routes
    authHandler := beaconmux.NewHandler(dbAdapter, sessionManager, &auth.Config{})
    authHandler.RegisterRoutes(r)

    // Protected Routes
    api := r.PathPrefix("/api").Subrouter()
    api.Use(beaconmux.RequireAuthJSON(sessionManager))
    api.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
        // ...
    }).Methods("GET")

    http.ListenAndServe(":8080", r)
}
```

## Middleware

- `SessionMiddleware`
- `RequireAuth`
- `TenantMiddleware`

## Helpers

Identical to Standard HTTP integration:

- `beaconmux.GetTenant(ctx)`
