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

Stores authentication accounts for OAuth providers, email/password credentials, etc.

Table Name: `accounts`

| Field                      | Type        | Description                                                |
| :------------------------- | :---------- | :--------------------------------------------------------- |
| `id`                       | `string`    | Unique identifier.                                         |
| `user_id`                  | `string`    | Foreign key to `users.id`.                                 |
| `account_id`               | `string`    | Provider's account ID (email for credentials, OAuth ID).   |
| `provider_id`              | `string`    | Provider ID ("local", "google", "github", etc.).           |
| `provider_type`            | `string`    | Provider type ("credential", "oauth").                     |
| `password`                 | `string`    | Hashed password (for credential accounts only).            |
| `access_token`             | `string`    | OAuth Access Token.                                        |
| `refresh_token`            | `string`    | OAuth Refresh Token.                                       |
| `access_token_expires_at`  | `timestamp` | Access token expiry.                                       |
| `refresh_token_expires_at` | `timestamp` | Refresh token expiry.                                      |
| `scope`                    | `string`    | OAuth scope.                                               |
| `id_token`                 | `string`    | OIDC ID Token.                                             |
| `created_at`               | `timestamp` | Creation time.                                             |
| `updated_at`               | `timestamp` | Last update time.                                          |

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

## Schema Migration

### Migrating from v0.6.0 and Earlier

If you're using an older schema with `provider` column instead of `provider_id`, you need to migrate:

**PostgreSQL Migration:**
```sql
-- Add new columns
ALTER TABLE accounts ADD COLUMN provider_id VARCHAR(255);
ALTER TABLE accounts ADD COLUMN provider_type VARCHAR(50);

-- Migrate data (adjust based on your provider values)
UPDATE accounts SET
  provider_id = CASE
    WHEN provider = 'credentials' THEN 'local'
    ELSE provider
  END,
  provider_type = CASE
    WHEN provider = 'credentials' THEN 'credential'
    WHEN password IS NOT NULL THEN 'credential'
    ELSE 'oauth'
  END;

-- Make columns required
ALTER TABLE accounts ALTER COLUMN provider_id SET NOT NULL;
ALTER TABLE accounts ALTER COLUMN provider_type SET NOT NULL;

-- Drop old column
ALTER TABLE accounts DROP COLUMN provider;

-- Recreate unique constraint
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_provider_account_id_key;
ALTER TABLE accounts ADD CONSTRAINT accounts_provider_account_id_unique UNIQUE(provider_id, account_id);
```

### Using the CLI Generator

The recommended way to create the correct schema is using the `beacon` CLI tool:

```bash
# Generate PostgreSQL schema
beacon generate --adapter postgres --id-type string

# Generate with UUID IDs
beacon generate --adapter postgres --id-type uuid

# Generate for other databases
beacon generate --adapter mysql --id-type string
beacon generate --adapter sqlite --id-type string
```

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
