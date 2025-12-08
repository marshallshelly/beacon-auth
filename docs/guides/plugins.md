---
title: Plugins
description: Guide to available authentication plugins.
---

BeaconAuth uses a plugin architecture. Features are opt-in.

## Available Plugins

### 1. Email & Password (`emailpassword`)

Provides classic email and password registration and login.
Included in `github.com/marshallshelly/beacon-auth/plugins/emailpassword`.

**Endpoints Added:**

- `POST /auth/register`: Create a new account.
- `POST /auth/login`: Authenticate and create a session.

### 2. Two-Factor Authentication (`twofa`)

Provides TOTP-based 2FA (compatible with Google Authenticator) with backup codes.
Included in `github.com/marshallshelly/beacon-auth/plugins/twofa`.

**Prerequisites:**

- Database must have `two_factors` and `two_factor_backup_codes` tables (see [getting started](../getting-started/quickstart/)).
- User model requires `two_factor_enabled` boolean.

**Endpoints Added:**

- `POST /auth/2fa/generate`: Generate a secret, QR code URI, and backup codes.
- `POST /auth/2fa/enable`: Verify code and enable 2FA.
- `POST /auth/2fa/verify`: Verify a TOTP code or backup code during login flows.
- `POST /auth/2fa/disable`: Disable 2FA and remove secrets.

### 3. OAuth (`oauth`)

Support for Social Login with multiple providers.
Included in `github.com/marshallshelly/beacon-auth/plugins/oauth`.

**Available Providers:**

#### GitHub

- Email verification status tracking
- Primary email detection
- Standard OAuth 2.0 flow

```go
import "github.com/marshallshelly/beacon-auth/plugins/oauth/providers"

githubProvider := providers.NewGitHub(
    "your-client-id",
    "your-client-secret",
    []string{"user:email"}, // scopes
)
```

#### Google

- PKCE (Proof Key for Code Exchange) support
- ID token handling
- Refresh token support (with `AccessType: "offline"`)

```go
googleProvider := providers.NewGoogle(&providers.GoogleOptions{
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret",
    AccessType:   "offline", // For refresh tokens
})
```

#### Discord

- Custom and default avatar URL generation
- Support for old discriminator and new username systems
- Animated avatar detection (GIF vs PNG)
- Refresh token support

```go
discordProvider := providers.NewDiscord(&providers.DiscordOptions{
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret",
    Prompt:       "none", // or "consent"
})
```

**Usage:**

```go
import (
    "github.com/marshallshelly/beacon-auth/plugins/oauth"
    "github.com/marshallshelly/beacon-auth/plugins/oauth/providers"
)

auth, _ := beaconauth.New(
    beaconauth.WithAdapter(adapter),
    beaconauth.WithPlugins(
        oauth.New(githubProvider, googleProvider, discordProvider),
    ),
)
```

**Endpoints Added:**

- `GET /auth/oauth/{provider}/login`: Redirect to provider (e.g., `/auth/oauth/github/login`).
- `GET /auth/oauth/{provider}/callback`: Handle callback (e.g., `/auth/oauth/github/callback`).
