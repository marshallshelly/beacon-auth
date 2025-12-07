package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/marshallshelly/beaconauth/core"
	"github.com/redis/go-redis/v9"
)

// RedisStore implements Store using Redis
type RedisStore struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// NewRedisStore creates a new Redis session store
func NewRedisStore(addr, password string, db int, prefix string, ttl time.Duration) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStore{
		client: client,
		prefix: prefix,
		ttl:    ttl,
	}, nil
}

// Get retrieves a session by token from Redis
func (r *RedisStore) Get(ctx context.Context, token string) (*core.Session, *core.User, error) {
	key := r.prefix + token

	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil, nil // Session not found
		}
		return nil, nil, fmt.Errorf("redis get error: %w", err)
	}

	var sessionData SessionData
	if err := json.Unmarshal(data, &sessionData); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	// Check if session is expired
	if time.Now().After(sessionData.Session.ExpiresAt) {
		r.Delete(ctx, token) // Clean up expired session
		return nil, nil, nil
	}

	return sessionData.Session, sessionData.User, nil
}

// Set stores a session in Redis
func (r *RedisStore) Set(ctx context.Context, session *core.Session) error {
	// We need the user data too, but for Redis store we'll store just the session
	// The session manager will handle fetching user data separately
	key := r.prefix + session.Token

	sessionData := SessionData{
		Session: session,
		// User will be fetched from DB when needed
	}

	data, err := json.Marshal(sessionData)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	// Calculate TTL until expiration
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("session already expired")
	}

	// Use the shorter of configured TTL or time until expiration
	if r.ttl > 0 && r.ttl < ttl {
		ttl = r.ttl
	}

	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	return nil
}

// SetWithUser stores a session with user data in Redis
func (r *RedisStore) SetWithUser(ctx context.Context, session *core.Session, user *core.User) error {
	key := r.prefix + session.Token

	sessionData := SessionData{
		Session: session,
		User:    user,
	}

	data, err := json.Marshal(sessionData)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	// Calculate TTL until expiration
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("session already expired")
	}

	// Use the shorter of configured TTL or time until expiration
	if r.ttl > 0 && r.ttl < ttl {
		ttl = r.ttl
	}

	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	return nil
}

// Delete removes a session from Redis
func (r *RedisStore) Delete(ctx context.Context, token string) error {
	key := r.prefix + token

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis delete error: %w", err)
	}

	return nil
}

// DeleteByUserID removes all sessions for a user from Redis
func (r *RedisStore) DeleteByUserID(ctx context.Context, userID string) error {
	// Scan for all session keys
	pattern := r.prefix + "*"

	var cursor uint64
	var keys []string

	for {
		var scanKeys []string
		var err error

		scanKeys, cursor, err = r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("redis scan error: %w", err)
		}

		keys = append(keys, scanKeys...)

		if cursor == 0 {
			break
		}
	}

	// Check each key to see if it belongs to the user
	for _, key := range keys {
		data, err := r.client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var sessionData SessionData
		if err := json.Unmarshal(data, &sessionData); err != nil {
			continue
		}

		if sessionData.Session.UserID == userID {
			r.client.Del(ctx, key)
		}
	}

	return nil
}

// Cleanup removes expired sessions from Redis
// Note: Redis automatically removes expired keys, so this is a no-op
func (r *RedisStore) Cleanup(ctx context.Context) error {
	return nil
}

// Close closes the Redis connection
func (r *RedisStore) Close() error {
	return r.client.Close()
}

// Ping checks the Redis connection
func (r *RedisStore) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
