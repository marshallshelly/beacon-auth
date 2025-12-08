---
title: Database
description: Learn how to use a database with BeaconAuth.
---

BeaconAuth uses a database (via an **Adapter**) to store user data, sessions, accounts, and verification tokens.

## Adapters

You can pass a supported database adapter to the `New` function.

```go
import (
    "github.com/beaconauth/auth"
    "github.com/beaconauth/auth/adapters/postgres"
)

// Initialize Postgres Adapter
adapter, _ := postgres.New(postgres.Config{
    URL: "postgres://user:pass@localhost:5432/db",
})

// Pass to BeaconAuth
auth := beaconauth.New(adapter, ...)
```

## Core Schema

BeaconAuth requires the following tables to be present in your database.

### User

Table Name: `user` (or configured name)

| Field            | Type        | Description                |
| :--------------- | :---------- | :------------------------- |
| `id`             | `string`    | Unique identifier.         |
| `name`           | `string`    | Display name.              |
| `email`          | `string`    | User's email address.      |
| `email_verified` | `boolean`   | Email verification status. |
| `image`          | `string`    | Avatar URL (optional).     |
| `role`           | `string`    | RBAC role (e.g. "admin").  |
| `banned`         | `boolean`   | Ban status.                |
| `ban_reason`     | `string`    | Reason for ban.            |
| `ban_expires`    | `timestamp` | Ban expiration.            |
| `created_at`     | `timestamp` | Creation time.             |
| `updated_at`     | `timestamp` | Last update time.          |

### Session

Table Name: `session`

| Field             | Type        | Description                   |
| :---------------- | :---------- | :---------------------------- |
| `id`              | `string`    | Unique identifier.            |
| `user_id`         | `string`    | Foreign key to `user.id`.     |
| `token`           | `string`    | Unique session token.         |
| `expires_at`      | `timestamp` | Session expiration.           |
| `ip_address`      | `string`    | Client IP (optional).         |
| `user_agent`      | `string`    | Client User Agent (optional). |
| `impersonated_by` | `string`    | Admin ID if impersonating.    |
| `created_at`      | `timestamp` | Creation time.                |
| `updated_at`      | `timestamp` | Last update time.             |

### Account

Legacy implementation for OAuth/Provider accounts.

Table Name: `account`

| Field                      | Type        | Description                               |
| :------------------------- | :---------- | :---------------------------------------- |
| `id`                       | `string`    | Unique identifier.                        |
| `user_id`                  | `string`    | Foreign key to `user.id`.                 |
| `account_id`               | `string`    | Provider's account ID.                    |
| `provider_id`              | `string`    | Provider ID (e.g. "google").              |
| `access_token`             | `string`    | OAuth Access Token.                       |
| `refresh_token`            | `string`    | OAuth Refresh Token.                      |
| `access_token_expires_at`  | `timestamp` | Access token expiry.                      |
| `refresh_token_expires_at` | `timestamp` | Refresh token expiry.                     |
| `scope`                    | `string`    | OAuth scope.                              |
| `id_token`                 | `string`    | OIDC ID Token.                            |
| `password`                 | `string`    | Hashed password (if credential provider). |
| `created_at`               | `timestamp` | Creation time.                            |
| `updated_at`               | `timestamp` | Last update time.                         |

### Verification

Table Name: `verification`

| Field        | Type        | Description                    |
| :----------- | :---------- | :----------------------------- |
| `id`         | `string`    | Unique identifier.             |
| `identifier` | `string`    | Email or value being verified. |
| `value`      | `string`    | The token/OTP value.           |
| `expires_at` | `timestamp` | Expiration time.               |
| `created_at` | `timestamp` | Creation time.                 |
| `updated_at` | `timestamp` | Last update time.              |

## Extending the Schema

You can add arbitrary columns to your database tables. BeaconAuth automatically captures these extra fields into a `Metadata` map on the Go structs (`User`, `Session`, `Account`).

**Example:**
If you add a `subscription_status` column to your `user` table, it will be available in `user.Metadata["subscription_status"]`.

## ID Generation

BeaconAuth generates unique string IDs (22-char URL-safe Base64) by default for all entities. ensuring compatibility across distributed systems.

## Database Hooks

The Adapter interface allows you to intercept calls. You can wrap the standard adapter with your own implementation to add `Before` or `After` hooks for `Create`, `Update`, or `Delete` operations.

```go
type MyHookAdapter struct {
    core.Adapter
}

func (h *MyHookAdapter) Create(ctx context.Context, model string, data map[string]interface{}) (interface{}, error) {
    // Before hook
    if model == "user" {
        // validate or modify data
    }

    // Call original
    return h.Adapter.Create(ctx, model, data)
}
```
