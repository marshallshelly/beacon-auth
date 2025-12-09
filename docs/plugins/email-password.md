---
title: Email & Password
description: Implementing email and password authentication with BeaconAuth.
---

Email and password authentication is the standard method for user accounts. BeaconAuth provides a robust, pre-built plugin for handling self-hosted email/password registration and login.

## Installation

To enable email and password authentication, import the plugin and add it to your `beaconauth.New` configuration.

```go title="main.go"
import (
    "github.com/marshallshelly/beacon-auth/plugins/emailpassword"
)

func main() {
    // ... setup adapter ...

    auth, err := beaconauth.New(
        beaconauth.WithAdapter(adapter),
        beaconauth.WithPlugins(
            emailpassword.New(), // Add the plugin here
        ),
    )
    if err != nil {
        log.Fatal(err)
    }

    // ...
}
```

## Usage

### Sign Up

To create a new account, send a `POST` request to `/auth/register`.

**Endpoint:** `POST /auth/register`

**Request Body:**

```json
{
  "email": "john.doe@example.com",
  "password": "secure-password-123",
  "name": "John Doe"
}
```

- **email** (required): The user's email address. Must be unique.
- **password** (required): The user's password. Must meet minimum length requirements (default: 8 characters).
- **name** (optional): The user's display name.

**Response:**

On success, the user is created, a session is started, and the user object is returned.

```json
{
  "id": "user_123...",
  "email": "john.doe@example.com",
  "name": "John Doe",
  "created_at": "..."
}
```

### Sign In

To authenticate an existing user, send a `POST` request to `/auth/login`.

**Endpoint:** `POST /auth/login`

**Request Body:**

```json
{
  "email": "john.doe@example.com",
  "password": "secure-password-123"
}
```

**Response:**

On success, a session cookie is set and the user object is returned.

```json
{
  "id": "user_123...",
  "email": "john.doe@example.com",
  ...
}
```

### Password Hashing

BeaconAuth uses `bcrypt` (via `golang.org/x/crypto/bcrypt`) or your configured password hasher to securely hash passwords before storing them. Plain text passwords are never stored in the database.

### Configuration

The default configuration requires a minimum password length of 8 characters. You can customize this in the main `auth.Config` struct passed to `beaconauth.New` (if using custom config options, although specific detailed config for this plugin might be exposed in future versions).

```go
type AuthConfig struct {
    EmailPassword struct {
        MinPasswordLength int // Default: 8
    }
}
```
