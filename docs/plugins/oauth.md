---
title: OAuth
description: Support for Social Login with multiple providers.
---

BeaconAuth provides a unified OAuth 2.0 implementation to let users log in with their existing accounts from Google, GitHub, Discord, and Apple.

## Installation

Add the `oauth` plugin to your `beaconauth` instance. You must configure each provider you wish to support.

```go title="main.go"
import (
    "github.com/marshallshelly/beacon-auth/plugins/oauth"
    "github.com/marshallshelly/beacon-auth/plugins/oauth/providers"
)

func main() {
    // ... setup adapter ...

    // Configure Google Provider
    googleProvider := providers.NewGoogle(&providers.GoogleOptions{
        ClientID:     "your-google-client-id",
        ClientSecret: "your-google-client-secret",
        AccessType:   "offline", // Request refresh token
        Prompt:       "consent", // Force consent screen
        RedirectURI:  "http://localhost:8080/auth/oauth/google/callback",
    })

    // Configure GitHub Provider
    githubProvider := providers.NewGitHub(
        "your-github-client-id",
        "your-github-client-secret",
        []string{"user:email"},
    )

    auth, _ := beaconauth.New(
        beaconauth.WithAdapter(adapter),
        beaconauth.WithPlugins(
            // Initialize OAuth plugin with all desired providers
            oauth.New(googleProvider, githubProvider),
        ),
    )
}
```

## Google Provider

To use Google as a social provider, you need to obtain OAuth 2.0 credentials from the Google Cloud Console.

### 1. Get Credentials

1. Go to the [Google Cloud Console](https://console.cloud.google.com/apis/credentials).
2. Click **Create Credentials** -> **OAuth client ID**.
3. Select **Web application** as the application type.
4. Add your **Authorized redirect URIs**:
   - Local: `http://localhost:8080/auth/oauth/google/callback`
   - Production: `https://your-domain.com/auth/oauth/google/callback`
5. Copy the **Client ID** and **Client Secret**.

### 2. Configuration Options

The `providers.GoogleOptions` struct supports several configurations:

- **ClientID**: Your public Client ID.
- **ClientSecret**: Your secret key.
- **Scopes**: Array of scopes to request. Default includes `email` and `profile`.
- **RedirectURI**: Must match exactly what you registered in Google Console.
- **AccessType**: Set to `"offline"` to receive a refresh token.
- **Prompt**: Set to `"consent"` to force the consent screen (useful for debugging or ensuring refresh tokens are returned).

### 3. Usage

Start the login flow by redirecting the user to:

`GET /auth/oauth/google/login`

BeaconAuth will handle the redirection to Google, the callback processing (`/auth/oauth/google/callback`), user creation (if new), and session creation.

## Other Providers

### GitHub

Uses standard OAuth 2.0 flow. Requires `client_id` and `client_secret`.  
Scopes defaults to `read:user` and `user:email` if not specified.

### Discord

Supports `discord.com/api` integration.

- Supports grabbing user avatar strings.
- Uses standard OAuth flow.

### Apple

Supports "Sign in with Apple" for iOS and Web.

- Requires `TeamID`, `KeyID`, `ClientID` (Service ID), and a `PrivateKey` (PEM format).
- Generates Client Secret (JWT) on the fly.

## Endpoints

The OAuth plugin automatically registers the following endpoints for _each_ configured provider:

- `GET /auth/oauth/{provider}/login`: Initiates the OAuth flow. Redirects user to the provider.
- `GET /auth/oauth/{provider}/callback`: The callback URL provider sends user back to. Exchanges code for tokens and logs user in.

Where `{provider}` is the provider's ID (e.g., `google`, `github`, `discord`, `apple`).
