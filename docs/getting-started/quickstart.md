---
title: Quickstart
description: Get up and running with BeaconAuth using Email/Password and 2FA.
---

BeaconAuth is a modular, plugin-based authentication library for Go.

## 📦 Installation

```bash
go get github.com/marshallshelly/beacon-auth
```

## 🛠️ Basic Setup

Create a simple `main.go` file to set up the server with Email/Password and 2FA support.

```go
package main

import (
	"log"
	"net/http"

	"github.com/marshallshelly/beacon-auth/beaconauth"
	"github.com/marshallshelly/beacon-auth/adapters/postgres"
	"github.com/marshallshelly/beacon-auth/plugins/emailpassword"
	"github.com/marshallshelly/beacon-auth/plugins/twofa"
)

func main() {
	// 1. Initialize Database Adapter
	dsn := "host=localhost user=postgres password=postgres dbname=auth port=5432 sslmode=disable"
	adapter, err := postgres.New(dsn)
	if err != nil {
		log.Fatal(err)
	}

	// 2. Configure BeaconAuth
	auth, err := beaconauth.New(
		beaconauth.WithAdapter(adapter),
		beaconauth.WithSecret("your-super-secret-key-at-least-32-bytes"),
		beaconauth.WithBaseURL("http://localhost:3000"),
		// Register Plugins
		beaconauth.WithPlugins(
			emailpassword.New(), // Adds /auth/register, /auth/login
			twofa.New(),         // Adds /auth/2fa/generate, /auth/enable
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer auth.Close()

	// 3. Mount Routes
	// The handler mounts all plugin endpoints under the BasePath (default: /auth)
	http.Handle("/auth/", auth.Handler())

	// 4. Start Server
	log.Println("Server starting on :3000...")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
```

## 🗄️ Database Schema

BeaconAuth requires the following tables. You can run this SQL to initialize your Postgres database:

```sql
-- Users Table
CREATE TABLE users (
    id VARCHAR(255) PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    email_verified BOOLEAN DEFAULT FALSE,
    name VARCHAR(255),
    image TEXT,
    two_factor_enabled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Sessions Table
CREATE TABLE sessions (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Accounts Table (for OAuth and Credentials)
CREATE TABLE accounts (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) REFERENCES users(id) ON DELETE CASCADE,
    account_id VARCHAR(255) NOT NULL,
    provider VARCHAR(255) NOT NULL,
    provider_type VARCHAR(50) NOT NULL, -- 'credential', 'oauth'
    password TEXT, -- Encrypted password for credentials
    access_token TEXT,
    refresh_token TEXT,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider, account_id)
);

-- Two Factor Secrets Table
CREATE TABLE two_factors (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) REFERENCES users(id) ON DELETE CASCADE,
    secret TEXT NOT NULL, -- In production, ensure this is encrypted!
    confirmed BOOLEAN DEFAULT FALSE,
    backup_codes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

## 🚀 Next Steps

- **Secure your app**: Set `WithSecret` from environment variables.
- **Add OAuth**: Use `beaconauth.WithPlugins(oauth.New(github.Provider(...)))`.
- **Customize**: Explore [Configuration](../reference/configuration/) for session and security settings.
