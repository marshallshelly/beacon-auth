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

Provides TOTP-based 2FA (compatible with Google Authenticator).
Included in `github.com/marshallshelly/beacon-auth/plugins/twofa`.

**Prerequisites:**

- Database must have `two_factors` table (see [getting started](../getting-started/quickstart/)).
- User model requires `two_factor_enabled` boolean.

**Endpoints Added:**

- `POST /auth/2fa/generate`: Generate a secret and QR code URI.
- `POST /auth/2fa/enable`: Verify code and enable 2FA.
- `POST /auth/2fa/disable`: Disable 2FA.

### 3. OAuth (`oauth`)

Support for Social Login (GitHub, etc.).
Included in `github.com/marshallshelly/beacon-auth/plugins/oauth`.

**Usage:**

```go
oauth.New(
    github.NewProvider(os.Getenv("GITHUB_ID"), os.Getenv("GITHUB_SECRET")),
)
```

**Endpoints Added:**

- `GET /auth/oauth/:provider`: Redirect to provider.
- `GET /auth/oauth/:provider/callback`: Handle callback.
