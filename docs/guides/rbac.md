---
title: Admin & RBAC
description: Comprehensive guide to Admin features, Role-Based Access Control, and User Management in BeaconAuth.
---

BeaconAuth provides robust, built-in support for administrative features, role-based access control (RBAC), and user management. This system allows you to manage user roles, ban users, handle impersonation, and extend your data model with custom fields.

## Schema Reference

BeaconAuth automatically supports the following fields on the core data models. You may need to add these columns to your database tables if you are using a SQL adapter.

### User Table

The `User` struct includes fields for RBAC and Ban management.

| Field         | Type        | Description                                                                                           |
| :------------ | :---------- | :---------------------------------------------------------------------------------------------------- |
| `role`        | `string`    | The user's role (e.g., "admin", "user"). Defaults to `""` (no role) or can be set by the application. |
| `banned`      | `boolean`   | Indicates whether the user is banned (`true` = banned).                                               |
| `ban_reason`  | `string`    | The reason for the ban.                                                                               |
| `ban_expires` | `timestamp` | When the ban expires. If `null`, the ban is permanent.                                                |
| `metadata`    | `json/map`  | Internal field for custom data (see Additional Fields).                                               |

### Session Table

The `Session` struct includes support for impersonation.

| Field             | Type       | Description                                                                     |
| :---------------- | :--------- | :------------------------------------------------------------------------------ |
| `impersonated_by` | `string`   | The ID of the admin user who created this session (if impersonation is active). |
| `metadata`        | `json/map` | Internal field for custom session data.                                         |

## Access Control

BeaconAuth offers a flexible access control system based on the `Role` field.

### Checking Roles

You can check a user's permissions by verifying their role. The `User` struct provides a convenient helper method:

```go
// Check if the user has the 'admin' role
if user.HasRole("admin") {
    // Grant access to restricted resource
}
```

### Implementing Permissions

While BeaconAuth provides the role storage, you can define your own permission logic. A common pattern is to map roles to a set of allowed actions.

```go
var RolePermissions = map[string][]string{
    "admin": {"create:user", "delete:user", "view:dashboard"},
    "user":  {"view:dashboard"},
}

func HasPermission(user *core.User, action string) bool {
    perms := RolePermissions[user.Role]
    for _, p := range perms {
        if p == action {
            return true
        }
    }
    return false
}
```

## User Management

Administrative actions usually involve updating the user record via the `DataManager` or Adapter.

### Assigning Roles

To assign a role to a user (e.g., promoting a user to admin), update the user's `Role` field.

```go
// Promote user to admin
data := map[string]interface{}{
    "role": "admin",
}
auth.DataManager.UpdateUser(ctx, userID, data)
```

### Banning Users

Banning a user prevents them from accessing the system (middleware should check the `Banned` status).

**Ban a user:**

```go
import "time"

// Ban user for 7 days
expires := time.Now().Add(7 * 24 * time.Hour)

data := map[string]interface{}{
    "banned":      true,
    "ban_reason":  "Violation of terms",
    "ban_expires": 	expires,
}
auth.DataManager.UpdateUser(ctx, userID, data)
```

**Unban a user:**

```go
data := map[string]interface{}{
    "banned":      false,
    "ban_reason":  "",
    "ban_expires": nil,
}
auth.DataManager.UpdateUser(ctx, userID, data)
```

### Impersonation

Impersonation allows an admin to log in as another user to view the system from their perspective. This creates a session for the target user but flags it with the admin's ID.

_Note: Implementation of the impersonation flow depends on your specific handler logic, but the data model supports it:_

```go
// When creating a session for impersonation
// (Conceptual example using direct adapter access)
sessionData := map[string]interface{}{
    "user_id": targetUserID,
    "impersonated_by": adminUserID,
    // ... other session fields
}
adapter.Create(ctx, "sessions", sessionData)
```

## Additional Fields (Metadata)

BeaconAuth allows you to store arbitrary data on `User`, `Account`, and `Session` models without changing the core Go structs. This is handled via the `Metadata` field.

### How it Works

Any column in your database table that **does not** match a standard field on the struct is automatically captured into the `Metadata` map (`map[string]interface{}`).

### Example Usage

**1. Database Migration**
Add a custom column to your users table:

```sql
ALTER TABLE users ADD COLUMN subscription_tier VARCHAR(50);
```

**2. Accessing Data**
When you fetch the user, the data is automatically available:

```go
user, _ := auth.FindUserByEmail(ctx, "jane@example.com")

if tier, ok := user.Metadata["subscription_tier"].(string); ok {
    fmt.Printf("User Tier: %s\n", tier)
}
```

**3. Storing Data**
To save custom data, simply pass it in the map when creating or updating a user:

```go
data := map[string]interface{}{
    "name": "Jane Doe",
    "subscription_tier": "premium", // Custom field
}
auth.DataManager.UpdateUser(ctx, userID, data)
```

This feature allows you to extend the BeaconAuth data model to fit your application's unique requirements effortlessly.
