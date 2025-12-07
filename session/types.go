package session

import (
	"context"
	"time"

	"github.com/marshallshelly/beaconauth/core"
)

// Store defines the interface for session storage backends
type Store interface {
	// Get retrieves a session by token
	Get(ctx context.Context, token string) (*core.Session, *core.User, error)

	// Set stores a session
	Set(ctx context.Context, session *core.Session) error

	// Delete removes a session by token
	Delete(ctx context.Context, token string) error

	// DeleteByUserID removes all sessions for a user
	DeleteByUserID(ctx context.Context, userID string) error

	// Cleanup removes expired sessions (optional, for stores that need it)
	Cleanup(ctx context.Context) error

	// Close closes the store connection
	Close() error
}

// Config holds session configuration
type Config struct {
	// Cookie settings
	CookieName     string
	CookieDomain   string
	CookiePath     string
	CookieSecure   bool
	CookieHTTPOnly bool
	CookieSameSite string // "strict", "lax", "none"

	// Session settings
	ExpiresIn      time.Duration
	UpdateAge      time.Duration // Update session timestamp if older than this
	AbsoluteExpiry bool          // If true, session expires regardless of activity

	// Storage layers (in order of priority)
	// Session lookup: Cookie → Redis → Database
	// Session write: All layers
	EnableCookieStore bool
	EnableRedisStore  bool
	EnableDBStore     bool

	// Redis configuration
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	RedisPrefix   string // Key prefix for Redis keys

	// Secret for signing cookies/tokens
	Secret string

	// Issuer for JWT tokens (if using cookie store)
	Issuer string
}

// DefaultConfig returns default session configuration
func DefaultConfig() *Config {
	return &Config{
		CookieName:        "beacon_session",
		CookiePath:        "/",
		CookieSecure:      true,
		CookieHTTPOnly:    true,
		CookieSameSite:    "lax",
		ExpiresIn:         7 * 24 * time.Hour, // 7 days
		UpdateAge:         24 * time.Hour,     // Update if older than 1 day
		AbsoluteExpiry:    false,
		EnableCookieStore: true,
		EnableRedisStore:  true,
		EnableDBStore:     true,
		RedisPrefix:       "beacon:session:",
		Issuer:            "beaconauth",
	}
}

// SessionData represents session data for storage
type SessionData struct {
	Session *core.Session
	User    *core.User
}

// Strategy defines the multi-layer storage strategy
type Strategy string

const (
	// StrategyRedisFirst uses Redis as primary, DB as fallback
	StrategyRedisFirst Strategy = "redis_first"

	// StrategyDBFirst uses DB as primary, Redis as cache
	StrategyDBFirst Strategy = "db_first"

	// StrategyCookieOnly uses only signed cookies (stateless)
	StrategyCookieOnly Strategy = "cookie_only"

	// StrategyRedisOnly uses only Redis (no DB persistence)
	StrategyRedisOnly Strategy = "redis_only"

	// StrategyDBOnly uses only database (no caching)
	StrategyDBOnly Strategy = "db_only"
)
