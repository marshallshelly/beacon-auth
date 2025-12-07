package session

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/marshallshelly/beacon-auth/core"
)

// Manager orchestrates multi-layer session storage
type Manager struct {
	config      *Config
	cookieStore *CookieStore
	redisStore  *RedisStore
	dbStore     *DBStore
	strategy    Strategy
}

// Config returns the session configuration
func (m *Manager) Config() *Config {
	return m.config
}

// NewManager creates a new session manager
func NewManager(config *Config, dbAdapter core.Adapter) (*Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	m := &Manager{
		config: config,
	}

	// Initialize cookie store if enabled
	if config.EnableCookieStore {
		m.cookieStore = NewCookieStore(config.Secret, config.Issuer)
	}

	// Initialize Redis store if enabled
	if config.EnableRedisStore && config.RedisAddr != "" {
		redisStore, err := NewRedisStore(
			config.RedisAddr,
			config.RedisPassword,
			config.RedisDB,
			config.RedisPrefix,
			config.ExpiresIn,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create Redis store: %w", err)
		}
		m.redisStore = redisStore
	}

	// Initialize DB store if enabled
	if config.EnableDBStore && dbAdapter != nil {
		m.dbStore = NewDBStore(dbAdapter)
	}

	// Determine strategy
	m.strategy = m.determineStrategy()

	return m, nil
}

// determineStrategy determines the storage strategy based on enabled stores
func (m *Manager) determineStrategy() Strategy {
	cookieEnabled := m.cookieStore != nil
	redisEnabled := m.redisStore != nil
	dbEnabled := m.dbStore != nil

	if cookieEnabled && !redisEnabled && !dbEnabled {
		return StrategyCookieOnly
	}
	if redisEnabled && !dbEnabled {
		return StrategyRedisOnly
	}
	if dbEnabled && !redisEnabled {
		return StrategyDBOnly
	}
	if redisEnabled && dbEnabled {
		return StrategyRedisFirst // Default when both are enabled
	}

	return StrategyDBOnly // Fallback
}

// Get retrieves a session using the multi-layer strategy
// Lookup order: Cookie → Redis → Database
func (m *Manager) Get(ctx context.Context, token string) (*core.Session, *core.User, error) {
	switch m.strategy {
	case StrategyCookieOnly:
		return m.getFromCookie(ctx, token)

	case StrategyRedisOnly:
		return m.getFromRedis(ctx, token)

	case StrategyDBOnly:
		return m.getFromDB(ctx, token)

	case StrategyRedisFirst:
		// Try Redis first
		session, user, err := m.getFromRedis(ctx, token)
		if err == nil && session != nil {
			return session, user, nil
		}

		// Fallback to database
		session, user, err = m.getFromDB(ctx, token)
		if err != nil || session == nil {
			return nil, nil, err
		}

		// Cache in Redis for next time
		if m.redisStore != nil {
			m.redisStore.SetWithUser(ctx, session, user)
		}

		return session, user, nil

	case StrategyDBFirst:
		// Try database first
		session, user, err := m.getFromDB(ctx, token)
		if err == nil && session != nil {
			// Cache in Redis
			if m.redisStore != nil {
				m.redisStore.SetWithUser(ctx, session, user)
			}
			return session, user, nil
		}

		// Fallback to Redis (shouldn't happen normally)
		return m.getFromRedis(ctx, token)

	default:
		return nil, nil, fmt.Errorf("unknown strategy: %v", m.strategy)
	}
}

// getFromCookie retrieves session from cookie store
func (m *Manager) getFromCookie(ctx context.Context, token string) (*core.Session, *core.User, error) {
	if m.cookieStore == nil {
		return nil, nil, nil
	}
	return m.cookieStore.Get(ctx, token)
}

// getFromRedis retrieves session from Redis store
func (m *Manager) getFromRedis(ctx context.Context, token string) (*core.Session, *core.User, error) {
	if m.redisStore == nil {
		return nil, nil, nil
	}
	return m.redisStore.Get(ctx, token)
}

// getFromDB retrieves session from database store
func (m *Manager) getFromDB(ctx context.Context, token string) (*core.Session, *core.User, error) {
	if m.dbStore == nil {
		return nil, nil, nil
	}
	return m.dbStore.Get(ctx, token)
}

// Create creates a new session and stores it in all enabled layers
func (m *Manager) Create(ctx context.Context, userID string, opts *core.SessionOptions) (*core.Session, *core.User, string, error) {
	// Calculate expiration
	expiresAt := time.Now().Add(m.config.ExpiresIn)
	if opts != nil && opts.ExpiresIn != nil {
		expiresAt = time.Now().Add(*opts.ExpiresIn)
	}

	// Generate session ID and token
	sessionID, err := generateID()
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to generate session ID: %w", err)
	}

	token, err := generateToken()
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	// Create session object
	now := time.Now()
	session := &core.Session{
		ID:        sessionID,
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if opts != nil {
		session.IPAddress = opts.IPAddress
		session.UserAgent = opts.UserAgent
	}

	// Get user data
	var user *core.User
	if m.dbStore != nil {
		user, err = m.dbStore.internal.FindUserByID(ctx, userID)
		if err != nil {
			return nil, nil, "", fmt.Errorf("failed to find user: %w", err)
		}
	}

	// Store in all enabled layers
	if m.dbStore != nil {
		if err := m.dbStore.Set(ctx, session); err != nil {
			return nil, nil, "", fmt.Errorf("failed to store session in database: %w", err)
		}
	}

	if m.redisStore != nil {
		if err := m.redisStore.SetWithUser(ctx, session, user); err != nil {
			return nil, nil, "", fmt.Errorf("failed to store session in Redis: %w", err)
		}
	}

	// Generate cookie token if cookie store is enabled
	var cookieToken string
	if m.cookieStore != nil {
		cookieToken, err = m.cookieStore.CreateToken(session, user)
		if err != nil {
			return nil, nil, "", fmt.Errorf("failed to create cookie token: %w", err)
		}
		token = cookieToken // Use cookie token as the primary token
	}

	return session, user, token, nil
}

// Update updates a session's expiration time
func (m *Manager) Update(ctx context.Context, session *core.Session) error {
	// Check if session needs updating based on UpdateAge
	if time.Since(session.UpdatedAt) < m.config.UpdateAge {
		return nil // No update needed
	}

	// Update expiration time
	if !m.config.AbsoluteExpiry {
		session.ExpiresAt = time.Now().Add(m.config.ExpiresIn)
	}
	session.UpdatedAt = time.Now()

	// Update in all enabled layers
	if m.dbStore != nil {
		if err := m.dbStore.Set(ctx, session); err != nil {
			return fmt.Errorf("failed to update session in database: %w", err)
		}
	}

	if m.redisStore != nil {
		if err := m.redisStore.Set(ctx, session); err != nil {
			return fmt.Errorf("failed to update session in Redis: %w", err)
		}
	}

	return nil
}

// Delete removes a session from all layers
func (m *Manager) Delete(ctx context.Context, token string) error {
	var lastErr error

	if m.dbStore != nil {
		if err := m.dbStore.Delete(ctx, token); err != nil {
			lastErr = err
		}
	}

	if m.redisStore != nil {
		if err := m.redisStore.Delete(ctx, token); err != nil {
			lastErr = err
		}
	}

	// Cookie deletion happens client-side

	return lastErr
}

// DeleteByUserID removes all sessions for a user from all layers
func (m *Manager) DeleteByUserID(ctx context.Context, userID string) error {
	var lastErr error

	if m.dbStore != nil {
		if err := m.dbStore.DeleteByUserID(ctx, userID); err != nil {
			lastErr = err
		}
	}

	if m.redisStore != nil {
		if err := m.redisStore.DeleteByUserID(ctx, userID); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// Cleanup removes expired sessions from all layers
func (m *Manager) Cleanup(ctx context.Context) error {
	var lastErr error

	if m.dbStore != nil {
		if err := m.dbStore.Cleanup(ctx); err != nil {
			lastErr = err
		}
	}

	if m.redisStore != nil {
		if err := m.redisStore.Cleanup(ctx); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// Close closes all store connections
func (m *Manager) Close() error {
	var lastErr error

	if m.dbStore != nil {
		if err := m.dbStore.Close(); err != nil {
			lastErr = err
		}
	}

	if m.redisStore != nil {
		if err := m.redisStore.Close(); err != nil {
			lastErr = err
		}
	}

	if m.cookieStore != nil {
		if err := m.cookieStore.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// Helper functions

func generateID() (string, error) {
	return generateRandomString(16)
}

func generateToken() (string, error) {
	return generateRandomString(32)
}

func generateRandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	if _, err := randomBytes(b); err != nil {
		return "", err
	}

	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}

	return string(b), nil
}

func randomBytes(b []byte) (int, error) {
	return rand.Read(b)
}
