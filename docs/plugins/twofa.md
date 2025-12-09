---
title: Two-Factor Authentication
description: Secure user accounts with TOTP-based 2FA.
---

`OTP` `TOTP` `Backup Codes`

Two-Factor Authentication (2FA) adds an extra layer of security by requiring a second form of verification (Time-based One-Time Password) in addition to the standard password. BeaconAuth's `twofa` plugin implements standard TOTP (compatible with Google Authenticator, Authy, etc.) and backup recovery codes.

## Installation

Add the `twofa` plugin to your configuration. Note that you must also have the `Session` middleware configured, as 2FA operations require an active session.

```go title="main.go"
import (
    "github.com/marshallshelly/beacon-auth/plugins/twofa"
)

func main() {
    // ... setup ...
    auth, _ := beaconauth.New(
        beaconauth.WithAdapter(adapter),
        beaconauth.WithPlugins(
            twofa.New(),
        ),
    )
    // ...
}
```

## Prerequisite: Database Schema

This plugin requires two additional tables: `two_factors` and `two_factor_backup_codes`. Ensure your database migration includes these (see [Concepts: Database](../concepts/database)).

## Usage Flow

### 1. Generate Secret (Setup)

To start setting up 2FA, the user must first request a new secret.

**Endpoint:** `POST /auth/2fa/generate`  
**Requires Session:** Yes

**Response:**

```json
{
  "secret": "JBSWY3DPEHPK3PXP",
  "totpURI": "otpauth://totp/MyApp:user@example.com?secret=JBSWY3DPEHPK3PXP...",
  "backupCodes": ["a1b2c3d4", "e5f6g7h8", ...]
}
```

- **totpURI**: Can be rendered as a QR code for the user to scan.
- **secret**: The raw secret key (can be displayed for manual entry).
- **backupCodes**: A set of one-time codes the user should save immediately.

### 2. Enable 2FA

After scanning the QR code, the user must provide a valid code to confirm and enable 2FA.

**Endpoint:** `POST /auth/2fa/enable`  
**Requires Session:** Yes

**Request:**

```json
{
  "code": "123456",
  "secret": "JBSWY3DPEHPK3PXP" // The secret received in step 1
}
```

**Response:** `{"success": true}`

User's `two_factor_enabled` flag is now set to `true`.

### 3. Verify Code (During Login)

When a user logs in (e.g., via `emailpassword`), you should check if `user.TwoFactorEnabled` is true. If so, prompt them for a 2FA code.

**Endpoint:** `POST /auth/2fa/verify`

**Request:**

```json
{
  "email": "user@example.com",
  "code": "123456" // TOTP code OR a backup code
}
```

**Response:**

On success, a fresh session is created and returned.

```json
{
  "success": true,
  "user": { ... }
}
```

### 4. Disable 2FA

Allows a logged-in user to disable 2FA.

**Endpoint:** `POST /auth/2fa/disable`  
**Requires Session:** Yes

**Response:** `{"success": true}`

## Backup Codes

When `generate` is called, a set of 10 backup codes is created. These are stored securely in the database.

- A backup code can be used in place of a TOTP code during verification.
- Once used, a backup code is deleted/invalidated.
- Users can re-generate codes by running the setup flow again (which rotates the secret and codes).
