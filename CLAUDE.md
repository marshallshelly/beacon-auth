# CLAUDE.md

This file provides guidance when working with code in this repository.

## Recent Updates (Dec 8, 2025)

### Directory Cleanup Completed ✅

The project structure has been cleaned up to remove all empty placeholder directories. The codebase now contains only implemented, tested features. This makes it easier to understand what's ready foruse vs what's planned for future development.

**What was removed:** 20+ empty directories including docs, examples, mailer, ratelimit, schema, validation, empty plugin implementations, and empty framework integrations.

**What remains:** Core infrastructure, adapters (memory + postgres), session management, crypto, auth handlers, middleware, plugin system foundation, and Fiber integration.

**Test Status:** 48/48 tests passing ✅ (PostgreSQL adapter tests require local database)

### Infrastructure & Tooling Updates (Latest) 🚀

**1. Go 1.24 Upgrade:**

- Project upgraded to Go 1.24 in `go.mod` and CI workflows.
- All dependencies updated to latest versions (`go get -u ./...`).
- Implemented modern Go 1.23+ iterator patterns in Fiber adapter (`header.All()`).

**2. Linting & Code Quality:**

- `golangci-lint` configured with v2 format.
- Added intelligent exclusions for test files (e.g., unchecked `Close()` deferals).
- Fixed staticcheck deprecation warnings.
- Currently passing with **0 lint issues**.

**3. CI/CD Pipeline:**

- **Cross-Platform Tests**: Separate jobs for Linux (with Postgres) vs macOS/Windows.
- **Reliability**: Increased integration test timeouts to 10s to accommodate slow Argon2 hashing on CI runners.
- **Configuration**: Updated workflows to use `check-latest: true` for cutting-edge Go versions.
- **Documentation**: Added `docs.yml` workflow to automatically build and deploy Starlight site to GitHub Pages on push.

**4. Clean Up & Final Verification:**

- Verified removal of placeholder directories (`docs`, `examples`, `mailer`, etc.).
- Confirmed project structure relies only on implemented features.
- **Final Status**: Codebase is clean, lint-free (0 issues), and all tests pass (48/48). Ready for Phase 2 (OAuth/Plugins).

### Phase 2: OAuth & Architecture (Complete) ✅

**1. OAuth Plugin System:**

- Implemented `plugins/oauth` with strict provider interface.
- **Features**: State-based CSRF protection, Account Linking (auto-link by email), Session creation on callback.
- **Providers Implemented** (3 total):
  - **GitHub Provider**: Full implementation including verified email fallback logic
  - **Google Provider**: PKCE support, ID token handling, refresh token support
  - **Discord Provider**: Custom/default avatar URLs, animated avatar detection, refresh token support

**2. Core Architecture Refactor (Dependency Injection):**

- **Problem**: Resolved cyclic dependency between `core`, `session`, and `adapter` packages.
- **Solution**: Introduced `SessionManagerFactory` and `DataManagerFactory` in `core.Config`.
- **Implementation**: `beaconauth.New` now orchestrates dependency wiring, injecting concrete implementations into `core`.
- **Result**: Cleaner architecture, testable `core` with mocks, and working OAuth flow.

**3. Test Updates**:

- Refactored `core` tests to use mock factories.
- Verified `go test ./core` passes cleanly.

### Phase 3: Regular Authentication (Email/Password) 🔐

**1. Credentials Support:**

- Implemented `plugins/emailpassword` handling `/auth/register` and `/auth/login`.
- **Security**: Uses Argon2id for password hashing via `crypto` package.
- **Integration**: extending `AuthContext` to support password hashing and credential account management.

**2. Core Updates:**

- Added `PasswordHasher` interface and factory support.
- Implemented `CreateCredentialAccount` in `InternalAdapter`.
- Implemented `CreateSession/GetSession` delegation in `core.Auth` (fixing stub methods).

### Phase 4: Two-Factor Authentication (Complete) ✅

**1. Plugin Fully Implemented:**

- Created `plugins/twofa` with complete TOTP support (`github.com/pquerna/otp`)
- Endpoints: `/2fa/generate`, `/2fa/enable`, `/2fa/disable`, `/2fa/verify`
- **Features**:
  - TOTP secret generation and QR code support
  - Backup codes generation (10 codes per user)
  - Login verification with TOTP or backup codes
  - Automatic backup code consumption
  - Secure storage of 2FA secrets and backup codes
  - Integration with User model (TwoFactorEnabled field)
- **Infrastructure**:
  - Added `UpdateUser` to DataManager and InternalAdapter
  - Added `TwoFactorEnabled` field to User type
  - Complete database integration for secrets and backup codes.

### Phase 5: RBAC & Admin Features (Complete) ✅

**1. Role-Based Access Control (RBAC):**

- Added `Role` field to User struct.
- Implemented `HasRole` helper method.
- Added User Banning support (`Banned`, `BanReason`, `BanExpires`).
- Added Impersonation support via `Session.ImpersonatedBy`.

**2. Schema Alignment & Extensibility:**

- **Better-Auth Compatibility**: Updated `Account` and `Verification` structs to strict matches of better-auth schema (renamed fields `Provider`->`ProviderID`, `Token`->`Value`, etc.).
- **Extensible Schema**: Added `Metadata` (map) to User, Session, and Account structs to automatically capture arbitrary database columns without code changes.
- **Adapter Logic**: Updated generic adapters to handle Metadata mapping and new field names.

**3. Documentation**:

- Added `docs/guides/rbac.md`: Comprehensive guide to Admin/RBAC features.
- Added `docs/concepts/database.md`: Detailed guide on Database Schema, Adapters, and Extensibility.
- Updated `www/astro.config.mjs` to include new documentation sections.

---

## Project Overview

BeaconAuth is a modular, plugin-based authentication library for Go inspired by better-auth. It's designed to work with any database (via adapters), any Go web framework (via integrations), and be extensible through plugins.

**Current Status**: Phase 1 (Core Infrastructure) complete. The project is in active development.

## Development Commands

### Testing

```bash
# Run all tests
go test ./... -v

# Run tests for specific package
go test ./core -v

# Run specific test
go test ./core -v -run TestNew

# Run tests with coverage
go test ./... -cover
```

### Building

```bash
# Build all packages
go build ./...

# Build specific package
go build ./core

# Verify module dependencies
go mod tidy
go mod verify
```

## Architecture

### Core Design Patterns

**1. Functional Options Pattern**
Configuration uses the functional options pattern throughout:

```go
auth, err := beaconauth.New(
    beaconauth.WithAdapter(adapter),
    beaconauth.WithSecret("secret"),
    beaconauth.WithPlugins(oauth.New(), twofa.New()),
)
```

All `With*` functions return `Option` which is `func(*Config) error`. This allows flexible, extensible configuration without breaking changes.

**2. Plugin Architecture**
Everything is designed as a plugin. Core features like OAuth, 2FA, etc. will be implemented as plugins that hook into the lifecycle.

Plugin interface:

```go
type Plugin interface {
    ID() string
    Init(ctx *AuthContext) error
}
```

Plugins can register:

- HTTP endpoints
- Lifecycle hooks (before/after requests)
- Database schema extensions
- Middleware

**3. Adapter Pattern**
Database operations are abstracted through the `Adapter` interface. All database-specific code lives in adapter implementations (postgres, mysql, mongodb, etc.).

The adapter uses a generic query builder with operators (=, !=, IN, etc.) that each adapter translates to its native query language.

**Adapter System Architecture:**

The adapter system has three layers:

1. **Core Adapter Interface** (`core/interfaces.go`) - Defines the contract all adapters must implement
2. **Adapter Factory** (`adapter/factory.go`) - Wraps custom adapters with automatic type transformations (JSON, dates, booleans, field name mapping)
3. **Internal Adapter** (`adapter/internal.go`) - Provides high-level operations (CreateUser, FindSessionWithUser, etc.) that abstract database details

**Type Transformations:**
The adapter factory handles databases with different capabilities:

- Databases without native JSON support: Serializes maps to JSON strings
- Databases without native date support: Converts time.Time to RFC3339 strings
- Databases without native boolean support: Converts bool to 0/1
- Field name mapping: Handles differences like MongoDB's `_id` vs standard `id`

**Query Building:**
All adapters translate `core.Query` structs to native queries:

```go
query := &core.Query{
    Model: "users",
    Where: []core.WhereClause{
        {Field: "email", Operator: core.OpEqual, Value: "test@example.com"},
        {Field: "created_at", Operator: core.OpGreaterThan, Value: time.Now()},
    },
    OrderBy: []core.OrderBy{
        {Field: "created_at", Desc: true},
    },
    Limit: 10,
    Offset: 0,
}
```

**Adapter Test Suite:**
The `adapter.TestSuite` provides comprehensive tests for adapter implementations:

- All CRUD operations
- All query operators (=, !=, >, <, >=, <=, LIKE, IN, NOT IN, IS NULL, IS NOT NULL)
- Transactions with rollback
- Limit and offset
- OrderBy (ascending and descending)
- Connection management (Ping, Close)

Use this to ensure your adapter behaves consistently.

**4. Session Management (Multi-Layer Storage)**
Session management in BeaconAuth uses a multi-layer architecture for optimal performance and flexibility.

**Session Storage Layers:**

1. **Cookie Store** - Stateless JWT-like tokens stored client-side

   - HMAC-signed payload with session and user data
   - No server-side storage required
   - Fast but can't be revoked server-side
   - Best for: Stateless APIs, serverless deployments

2. **Redis Store** - Fast in-memory cache for active sessions

   - Automatic TTL management
   - Sub-millisecond lookups
   - Stores session + user data for quick access
   - Best for: High-traffic applications, microservices

3. **Database Store** - Persistent storage for all sessions
   - Uses adapter system (works with any database)
   - Survives Redis restarts
   - Can query/revoke sessions by user ID
   - Best for: Long-term session management, audit logs

**Multi-Layer Strategies:**

```go
// Redis-First (default): Redis cache with DB fallback
// Lookup: Redis → DB → cache in Redis
// Write: DB + Redis
StrategyRedisFirst

// DB-First: Database primary with Redis caching
// Lookup: DB → cache in Redis
// Write: DB + Redis
StrategyDBFirst

// Cookie-Only: Stateless, no server-side storage
// Lookup: Verify signature
// Write: Generate signed token
StrategyCookieOnly

// Redis-Only: No persistence (sessions lost on restart)
// Lookup: Redis only
// Write: Redis only
StrategyRedisOnly

// DB-Only: No caching (slower but simpler)
// Lookup: DB only
// Write: DB only
StrategyDBOnly
```

**Session Configuration:**

```go
config := &session.Config{
    // Cookie settings
    CookieName:     "beacon_session",
    CookieSecure:   true,
    CookieHTTPOnly: true,
    CookieSameSite: "lax",

    // Session lifetime
    ExpiresIn:      7 * 24 * time.Hour, // 7 days
    UpdateAge:      24 * time.Hour,     // Refresh if older than 1 day
    AbsoluteExpiry: false,              // Extend on activity

    // Enable storage layers
    EnableCookieStore: true,
    EnableRedisStore:  true,
    EnableDBStore:     true,

    // Redis connection
    RedisAddr:     "localhost:6379",
    RedisPassword: "",
    RedisDB:       0,
    RedisPrefix:   "beacon:session:",

    // Security
    Secret: "your-secret-key",
    Issuer: "beaconauth",
}

manager, err := session.NewManager(config, dbAdapter)
```

**Session Operations:**

```go
// Create session
session, user, token, err := manager.Create(ctx, userID, &core.SessionOptions{
    IPAddress: req.RemoteAddr,
    UserAgent: req.UserAgent(),
    ExpiresIn: &customDuration,
})

// Get session (checks all layers in order)
session, user, err := manager.Get(ctx, token)

// Update session expiration
err := manager.Update(ctx, session)

// Delete single session
err := manager.Delete(ctx, token)

// Delete all user sessions (logout from all devices)
err := manager.DeleteByUserID(ctx, userID)

// Cleanup expired sessions
err := manager.Cleanup(ctx)
```

**Session Security:**

- HMAC-SHA256 signatures for cookie tokens
- Automatic expiration enforcement
- Time-based session refresh (UpdateAge)
- Absolute expiration option (no extension)
- IP and User-Agent tracking
- Secure, HTTPOnly, SameSite cookie attributes

**Session Middleware:**

```go
// Load session into context
app.Use(middleware.SessionMiddleware(sessionManager))

// Protect routes requiring authentication
protectedRoutes.Use(middleware.RequireAuth(sessionManager))

// API routes with JSON error responses
api.Use(middleware.RequireAuthJSON(sessionManager))

// Access session and user in handlers
func handler(w http.ResponseWriter, r *http.Request) {
    session := core.GetSession(r.Context())
    user := core.GetUser(r.Context())
    // ...
}
```

**5. Password Hashing and Crypto**
BeaconAuth uses Argon2id for password hashing - the recommended algorithm for password storage.

**Password Hashing:**

```go
hasher := crypto.NewArgon2Hasher()

// Hash a password
hash, err := hasher.Hash("user-password")

// Verify a password
valid, err := hasher.Verify("user-password", hash)
```

**Argon2id Parameters:**

- Memory: 64 MB
- Iterations: 3
- Parallelism: 2
- Salt length: 16 bytes
- Key length: 32 bytes

**Security Features:**

- Cryptographically secure random salt generation
- Constant-time password comparison (prevents timing attacks)
- PHC string format for hash encoding
- Configurable parameters for future-proofing

**Token Generation:**

```go
// Generate secure tokens
token, err := crypto.GenerateID()
sessionToken, err := crypto.GenerateSessionToken()
verifyToken, err := crypto.GenerateVerificationToken()

// Custom token generation
generator := crypto.NewTokenGenerator()
token, err := generator.Generate(32) // 32 bytes, base64-encoded
hexToken, err := generator.GenerateHex(16) // 16 bytes, hex-encoded
```

**6. Context-Driven**
Request-scoped data flows through `context.Context` using typed keys:

- `GetSession(ctx)` - retrieve current session
- `GetUser(ctx)` - retrieve current user
- `GetAuthContext(ctx)` - retrieve auth configuration

Never pass data through global variables.

**7. Email/Password Authentication Handlers**
BeaconAuth provides ready-to-use HTTP handlers for email/password authentication.

**Handler Setup:**

```go
import (
    "github.com/marshallshelly/beacon-auth/auth"
    "github.com/marshallshelly/beacon-auth/session"
)

// Create handler
handler := auth.NewHandler(dbAdapter, sessionManager, &auth.Config{
    MinPasswordLength:   8,
    RequireVerification: false,
    AllowSignup:         true,
})

// Use with standard http.Handler
http.HandleFunc("/auth/signup", handler.SignUp)
http.HandleFunc("/auth/signin", handler.SignIn)
http.HandleFunc("/auth/signout", handler.SignOut)
http.HandleFunc("/auth/session", handler.GetSession)
```

**Handler Features:**

- **SignUp**: Creates user, hashes password with Argon2id, creates session
- **SignIn**: Validates credentials, checks email verification, creates session
- **SignOut**: Deletes session and clears cookie
- **GetSession**: Returns current session and user from context

**Request/Response Format:**

```go
// SignUp/SignIn Request
{
    "email": "user@example.com",
    "password": "secure-password",
    "name": "User Name" // optional, signup only
}

// Auth Response
{
    "user": {
        "id": "user-id",
        "email": "user@example.com",
        "name": "User Name",
        "email_verified": false,
        "created_at": "2024-01-01T00:00:00Z"
    },
    "session": {
        "id": "session-id",
        "user_id": "user-id",
        "expires_at": "2024-01-08T00:00:00Z"
    },
    "token": "session-token"
}

// Error Response
{
    "error": "invalid_credentials",
    "message": "Invalid email or password"
}
```

**8. Fiber Framework Integration**
BeaconAuth includes first-class support for the Fiber web framework with multi-tenant capabilities.

**Basic Fiber Setup:**

```go
import (
    "github.com/gofiber/fiber/v2"
    beaconauth_fiber "github.com/marshallshelly/beacon-auth/integrations/fiber"
)

app := fiber.New()

// Add session middleware
app.Use(beaconauth_fiber.SessionMiddleware(sessionManager))

// Create auth handler
authHandler := beaconauth_fiber.NewHandler(dbAdapter, sessionManager, &auth.Config{
    MinPasswordLength:   8,
    RequireVerification: false,
    AllowSignup:         true,
})

// Auth routes
app.Post("/auth/signup", authHandler.SignUp)
app.Post("/auth/signin", authHandler.SignIn)
app.Post("/auth/signout", authHandler.SignOut)
app.Get("/auth/session", authHandler.GetSession)

// Protected routes
protected := app.Group("/api")
protected.Use(beaconauth_fiber.RequireAuthJSON(sessionManager))

protected.Get("/profile", func(c *fiber.Ctx) error {
    user := beaconauth_fiber.GetUser(c)
    session := beaconauth_fiber.GetSession(c)

    return c.JSON(fiber.Map{
        "user": user,
        "session": session,
    })
})
```

**Multi-Tenant Fiber Setup:**

```go
// Tenant extraction middleware
tenantConfig := &beaconauth_fiber.TenantConfig{
    BaseDomain:   "example.com",    // sunnyview.example.com -> "sunnyview"
    TenantHeader: "X-Tenant-ID",        // Alternative: check header
    DefaultTenant: "",                   // Fallback tenant
}

app.Use(beaconauth_fiber.TenantMiddleware(tenantConfig))

// Tenant-specific database adapter
app.Use(beaconauth_fiber.TenantIsolationMiddleware(func(tenantID string) (interface{}, error) {
    // Connect to tenant-specific database
    return postgres.New(ctx, &postgres.Config{
        Database: "tenant_" + tenantID,
        // ... other config
    })
}))

// Access tenant and adapter in routes
app.Get("/api/data", func(c *fiber.Ctx) error {
    tenant := beaconauth_fiber.GetTenant(c)
    adapter := beaconauth_fiber.GetAdapter(c).(core.Adapter)

    // Query tenant-specific data
    users, _ := adapter.FindMany(c.Context(), &core.Query{
        Model: "users",
    })

    return c.JSON(fiber.Map{
        "tenant": tenant,
        "users": users,
    })
})
```

**Tenant Extraction Functions:**

- `ExtractTenantFromHost(hostname, baseDomain)` - Extract subdomain from hostname
- `TenantMiddleware(config)` - Middleware to extract and store tenant
- `GetTenant(c)` - Retrieve tenant from Fiber context
- `RequireTenant()` - Middleware requiring tenant presence
- `TenantIsolationMiddleware(getTenantAdapter)` - Load tenant-specific adapter

**Fiber Middleware:**

- `SessionMiddleware(manager)` - Load session into Fiber locals
- `RequireAuth(manager)` - Redirect to signin if no session (HTML)
- `RequireAuthJSON(manager)` - Return JSON error if no session (API)

**Context Helpers:**

- `GetSession(c)` - Retrieve session from Fiber context
- `GetUser(c)` - Retrieve user from Fiber context
- `GetUserID(c)` - Retrieve user ID from Fiber context
- `GetTenant(c)` - Retrieve tenant ID from Fiber context
- `GetAdapter(c)` - Retrieve adapter from Fiber context

### Package Structure

**`core/`** - Core implementation

- `auth.go` - Main Auth interface and implementation
- `types.go` - Core types (User, Session, Account, Verification)
- `config.go` - Configuration system with functional options
- `context.go` - Context management and helpers
- `interfaces.go` - All interfaces (Adapter, Plugin, Logger, etc.)
- `errors.go` - Error types and error codes
- `logger.go` - Logging interface and default implementation

**`beaconauth.go`** - Public API
Re-exports core types and functions. Users import `github.com/marshallshelly/beacon-auth`, not `/core`.

**Adapter directories** (Phase 2 complete):

- `adapter/` - Adapter factory, internal adapter, and test suite
- `adapters/memory/` - In-memory adapter for testing
- `adapters/postgres/` - PostgreSQL adapter with pgx
- `adapters/mysql/` - MySQL adapter with go-sql-driver
- `adapters/sqlite/` - SQLite adapter (pure Go) with modernc.org/sqlite
- `adapters/mssql/` - MSSQL adapter with microsoft/go-mssqldb

**Session directory** (Phase 3 complete):

- `session/` - Multi-layer session management system
  - `types.go` - Session types and configuration
  - `redis_store.go` - Redis-backed session storage
  - `db_store.go` - Database-backed session storage
  - `cookie_store.go` - Stateless cookie-based sessions
  - `manager.go` - Session manager with multi-layer orchestration

**Middleware directory** (Phase 3 complete):

- `middleware/` - HTTP middleware
  - `session.go` - Session loading and authentication middleware

**Crypto directory** (Phase 4 complete):

- `crypto/` - Cryptographic utilities
  - `hash.go` - Argon2id password hashing
  - `token.go` - Secure token generation
  - `hash_test.go` - Password hashing tests

**Auth directory** (Phase 4 complete):

- `auth/` - HTTP authentication handlers
  - `handlers.go` - SignUp, SignIn, SignOut, GetSession handlers
  - `handlers_test.go` - Comprehensive handler tests (14/14 passing)

**Integration directories** (Fiber complete):

- `integrations/fiber/` - Fiber framework integration
  - `integrations/chi/` - Chi framework integration
  - `integrations/gin/` - Gin framework integration
  - `integrations/echo/` - Echo framework integration
  - `integrations/mux/` - Gorilla Mux framework integration
  - `integrations/http/` - Standard net/http integration

**Empty directories** (planned for future phases):

- `plugins/` - Plugin implementations (oauth, twofa, etc.)

### Key Architectural Decisions

**Why functional options?**
Go idiom for extensible configuration. Better than config structs because:

- Backward compatible when adding new options
- Clear at call site what's being configured
- Easy to provide defaults

**Why context.Context for request data?**
Standard Go pattern for request-scoped values. Avoids global state and thread-local storage.

**Why interface-first?**
All major components (Adapter, Plugin, Logger, Mailer) are interfaces. This allows:

- Easy testing with mocks
- User-provided implementations
- No vendor lock-in

**Layered architecture:**

```
beaconauth.go (public API)
    ↓
core/ (implementation)
    ↓
adapter/ (database abstraction)
    ↓
adapters/* (database-specific)
```

### Testing Strategy

**Mock Adapter Pattern:**
Tests use a `mockAdapter` that implements the Adapter interface. See `core/auth_test.go` for the pattern.

When writing tests:

1. Create minimal mock that implements required interfaces
2. Use table-driven tests for multiple scenarios
3. Test error cases, not just happy path
4. Always call `auth.Close()` in tests to clean up resources

## Implementation Notes

### Adding a New Plugin

1. Create package in `plugins/yourplugin/`
2. Implement `Plugin` interface:

   ```go
   type YourPlugin struct {}

   func (p *YourPlugin) ID() string {
       return "yourplugin"
   }

   func (p *YourPlugin) Init(ctx *core.AuthContext) error {
       // Register endpoints, hooks, schema
       return nil
   }
   ```

3. Export constructor: `func New() *YourPlugin`
4. Users add with: `beaconauth.WithPlugins(yourplugin.New())`

### Adding a New Adapter

1. Create package in `adapters/yourdb/`
2. Implement `CustomAdapter` interface from `adapter/factory.go`
3. Handle Query translation (Where clauses, Joins, OrderBy)
4. Implement transaction support or fallback to sequential
5. Use the adapter test suite to verify correct behavior:

   ```go
   import (
       "github.com/marshallshelly/beacon-auth/adapter"
       "github.com/marshallshelly/beacon-auth/adapters/yourdb"
   )

   func TestYourDBAdapter(t *testing.T) {
       dbAdapter := yourdb.New(/* config */)

       suite := &adapter.TestSuite{
           Adapter: dbAdapter,
           SetupFunc: func(t *testing.T, a core.Adapter) {
               // Create test tables/collections
           },
           TeardownFunc: func(t *testing.T, a core.Adapter) {
               // Clean up and close
               a.Close()
           },
       }

       suite.RunAll(t)
   }
   ```

6. Optionally wrap with `adapter.Factory` for automatic type transformations

### Error Handling

Use `AuthError` for application errors:

```go
return nil, core.NewAuthError(
    core.ErrCodeUserNotFound,
    "user not found",
    err,
).WithDetails("email", email)
```

Predefined errors in `core/errors.go` should be used when applicable.

### Logging

Default logger logs to stdout with `[BeaconAuth]` prefix. Users can provide custom logger via `beaconauth.WithLogger()`.

Log levels: Debug, Info, Warn, Error. Use appropriately:

- Debug: Detailed diagnostic info
- Info: Normal operations (e.g., "BeaconAuth initialized")
- Warn: Recoverable issues
- Error: Errors that require attention

## Project Roadmap

Implemented (Phase 1):

- ✅ Core types and interfaces
- ✅ Configuration system
- ✅ Context management
- ✅ Error handling
- ✅ Logging

Implemented (Phase 2):

- ✅ Adapter factory with automatic type transformations
- ✅ Internal adapter for high-level database operations
- ✅ Memory adapter for testing (complete with OrderBy support)
- ✅ PostgreSQL adapter with connection pooling and transactions
- ✅ Adapter test suite for consistent behavior across adapters

Implemented (Phase 3):

- ✅ Session Store interface for pluggable storage backends
- ✅ Redis session store with automatic TTL
- ✅ Database session store using adapter system
- ✅ Cookie session store with signed JWT-like tokens (stateless)
- ✅ Multi-layer session manager with configurable strategies
- ✅ Session expiration and cleanup
- ✅ Session middleware (SessionMiddleware, RequireAuth, RequireAuthJSON)
- ✅ Comprehensive session tests (7/7 passing)

Implemented (Phase 4 - Complete):

- ✅ Argon2id password hashing with secure defaults
- ✅ Cryptographically secure token generation
- ✅ Password hasher interface for extensibility
- ✅ Crypto utilities test suite (4/4 passing)
- ✅ Email/password authentication handlers (SignUp, SignIn, SignOut, GetSession)
- ✅ Request validation and error handling
- ✅ Auth handler tests (14/14 passing)

Implemented (Fiber Integration - Complete):

- ✅ Fiber middleware for session management
- ✅ Fiber auth handlers wrapping core handlers
- ✅ Multi-tenant support with subdomain extraction
- ✅ Tenant isolation middleware for per-tenant databases
- ✅ Request/response adapters for Fiber ↔ net/http conversion
- ✅ Comprehensive integration tests (13/13 passing)
- ✅ Example code for basic and multi-tenant setups

**Current Architecture (As of Dec 8, 2025):**
The project has been cleaned up to focus on core, implemented features only. Empty placeholder directories for future features have been removed to maintain a clean codebase.

**Active Directories:**

- `core/` - Core types, interfaces, and configuration
- `adapter/` - Adapter factory and internal high-level operations
- `adapters/memory/` - In-memory adapter for testing
- `adapters/postgres/` - PostgreSQL adapter with pgx
- `adapters/mongodb/` - MongoDB adapter using mongo-driver
- `session/` - Multi-layer session management
- `middleware/` - Session and authentication middleware
- `crypto/` - Password hashing and token generation
- `auth/` - HTTP authentication handlers
- `integrations/fiber/` - Fiber framework integration with multi-tenant support
- `plugin/` - Plugin system foundation (manager, hooks, registry, lifecycle)

**Removed Directories (Planned for Future Phases):**

- Empty plugin implementations (oauth, twofa, magiclink, username)
- Empty framework integrations (chi, gin, echo, mux, stdlib)
- Empty adapter implementations (mysql, sqlite)
- Utility directories (docs, examples, internal, mailer, ratelimit, schema, validation)

**Next Development Priorities:**

1. Complete plugin system foundation (hooks, lifecycle, registry)
2. Add OAuth plugin as first real plugin implementation
3. Implement additional database adapters (MySQL, SQLite)
4. Add framework integrations (Chi, Gin, Echo)
5. Build rate limiting middleware
6. Create schema and migration system
7. Add validation utilities
8. Develop example applications
9. Create documentation website with Astro

See `/Users/marshallshelly/Documents/GitHub/auth/BEACONAUTH_IMPLEMENTATION_PLAN.md` for full roadmap.
