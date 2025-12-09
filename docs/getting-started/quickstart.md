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
    "context"
    "log"
    "net/http"

    "github.com/marshallshelly/beacon-auth/beaconauth"
    "github.com/marshallshelly/beacon-auth/adapters/postgres"
    "github.com/marshallshelly/beacon-auth/plugins/emailpassword"
    "github.com/marshallshelly/beacon-auth/plugins/twofa"
)

func main() {
    // 1. Initialize Database Adapter (PostgreSQL)
    adapter, err := postgres.New(context.Background(), &postgres.Config{
        Host:     "localhost",
        Port:     5432,
        Database: "auth",
        Username: "postgres",
        Password: "postgres",
    })
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
            twofa.New(),         // Adds /auth/2fa/generate, /auth/enable, /auth/2fa/verify, /auth/2fa/disable
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

BeaconAuth requires the following tables. You can easily generate the SQL schema using our CLI.

### Option A: Using the CLI (Recommended)

Generate the SQL schema for your specific database configuration:

```bash
# Generate schema for Postgres with 2FA support
go run github.com/marshallshelly/beacon-auth/cmd/beacon@latest generate \
  --adapter postgres \
  --plugins twofa \
  > schema.sql

# Run it against your database
psql $DATABASE_URL < schema.sql
```

See the [CLI Documentation](/beacon-auth/concepts/cli) for more options, including UUID and Serial ID support.

### Option B: Manual Setup

If you prefer to copy-paste the SQL manually:

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
    password_hash TEXT, -- Encrypted password hash for credential login
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
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Two Factor Backup Codes Table
CREATE TABLE two_factor_backup_codes (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) REFERENCES users(id) ON DELETE CASCADE,
    code VARCHAR(50) NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, code)
);
```

## 🚀 Next Steps

- **Secure your app**: Set `WithSecret` from environment variables.
- **Add OAuth**: See OAuth Providers section below.
- **Customize**: Explore [Configuration](../reference/configuration/) for session and security settings.

## 🌐 OAuth Providers

BeaconAuth supports multiple OAuth providers out of the box:

```go
import (
    "github.com/marshallshelly/beacon-auth/plugins/oauth"
    "github.com/marshallshelly/beacon-auth/plugins/oauth/providers"
)

// GitHub
githubProvider := providers.NewGitHub(
    "your-github-client-id",
    "your-github-client-secret",
    []string{"user:email"}, // scopes
)

// Google (with PKCE support)
googleProvider := providers.NewGoogle(&providers.GoogleOptions{
    ClientID:     "your-google-client-id",
    ClientSecret: "your-google-client-secret",
    AccessType:   "offline", // Request refresh token
})

// Discord
discordProvider := providers.NewDiscord(&providers.DiscordOptions{
    ClientID:     "your-discord-client-id",
    ClientSecret: "your-discord-client-secret",
    Prompt:       "none", // or "consent"
})

// Apple (Sign in with Apple)
appleProvider := providers.NewApple(&providers.AppleOptions{
    ClientID:   "your-service-id",
    TeamID:     "your-team-id",
    KeyID:      "your-key-id",
    PrivateKey: "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----",
})

// Add to BeaconAuth
auth, _ := beaconauth.New(
    beaconauth.WithAdapter(adapter),
    beaconauth.WithSecret("your-secret"),
    beaconauth.WithBaseURL("http://localhost:3000"),
    beaconauth.WithPlugins(
        oauth.New(githubProvider, googleProvider, discordProvider, appleProvider),
        emailpassword.New(),
        twofa.New(),
    ),
)

// OAuth endpoints will be available at:
// - /auth/oauth/github/login
// - /auth/oauth/github/callback
// - /auth/oauth/google/login
// - /auth/oauth/google/callback
// - /auth/oauth/discord/login
// - /auth/oauth/discord/callback
```

> MongoDB users: Use equivalent collections (`users`, `sessions`, `accounts`, `two_factors`). Field names are the same; indexes on `users.email`, `sessions.token`, and unique (`provider`, `account_id`) are recommended.

## 🔌 Using MongoDB (Alternative Adapter)

```go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/marshallshelly/beacon-auth/beaconauth"
    "github.com/marshallshelly/beacon-auth/adapters/mongodb"
)

func main() {
    // Initialize MongoDB adapter
    adapter, err := mongodb.New(context.Background(), &mongodb.Config{
        URI:      "mongodb://localhost:27017",
        Database: "auth",
    })
    if err != nil { log.Fatal(err) }
    defer adapter.Close()

    auth, err := beaconauth.New(
        beaconauth.WithAdapter(adapter),
        beaconauth.WithSecret("your-super-secret-key-at-least-32-bytes"),
        beaconauth.WithBaseURL("http://localhost:3000"),
    )
    if err != nil { log.Fatal(err) }
    defer auth.Close()

    http.Handle("/auth/", auth.Handler())
    log.Fatal(http.ListenAndServe(":3000", nil))
}
```
