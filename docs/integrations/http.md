---
title: Standard net/http Integration
description: Complete guide to integrating BeaconAuth with standard Go net/http
---

BeaconAuth provides integration with the standard [net/http](https://golang.org/pkg/net/http/) package, making it compatible with any router that supports standard `http.Handler` (like Chi, Gorilla Mux, Justinas Alice, etc.).

## Installation

```bash
go get github.com/marshallshelly/beacon-auth
```

## Basic Setup

### 1. Create Handlers

```go
package main

import (
    "log"
    "net/http"
    "time"

    "github.com/marshallshelly/beacon-auth/adapters/memory"
    "github.com/marshallshelly/beacon-auth/auth"
    beaconhttp "github.com/marshallshelly/beacon-auth/integrations/http"
    "github.com/marshallshelly/beacon-auth/session"
)

func main() {
    // Setup adapter and manager
    dbAdapter := memory.New()
    sessionManager, _ := session.NewManager(&session.Config{
        CookieName: "session",
        Secret:     "secret-key-at-least-32-bytes-long",
    }, dbAdapter)

    // Create handler
    authHandler := beaconhttp.NewHandler(dbAdapter, sessionManager, &auth.Config{})

    // Routes
    http.HandleFunc("/auth/signup", authHandler.SignUp)
    http.HandleFunc("/auth/signin", authHandler.SignIn)
    http.HandleFunc("/auth/signout", authHandler.SignOut)

    // Middleware
    middleware := beaconhttp.SessionMiddleware(sessionManager)

    // Wrap handler
    handler := middleware(http.DefaultServeMux)

    log.Fatal(http.ListenAndServe(":8080", handler))
}
```

## Middleware

### SessionMiddleware

Wraps an `http.Handler` and populates the request context with session data.

```go
finalHandler := beaconhttp.SessionMiddleware(sessionManager)(myHandler)
```

### RequireAuth / RequireAuthJSON

Enforces authentication for downstream handlers.

```go
protected := beaconhttp.RequireAuth(sessionManager)(protectedHandler)
```

## Multi-Tenant Support

```go
tenantConfig := &beaconhttp.TenantConfig{BaseDomain: "example.com"}
handler = beaconhttp.TenantMiddleware(tenantConfig)(handler)
```

Access tenant in handlers:

```go
tenant := beaconhttp.GetTenant(r.Context())
```
