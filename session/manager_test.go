package session

import (
	"context"
	"testing"
	"time"

	"github.com/marshallshelly/beaconauth/adapters/memory"
	"github.com/marshallshelly/beaconauth/core"
)

func TestManager_CreateAndGet(t *testing.T) {
	adapter := memory.New()
	defer adapter.Close()

	config := DefaultConfig()
	config.EnableRedisStore = false // Disable Redis for this test
	config.EnableCookieStore = false

	manager, err := NewManager(config, adapter)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Create a user first
	user := map[string]interface{}{
		"id":    "user1",
		"email": "test@example.com",
		"name":  "Test User",
	}
	adapter.Create(ctx, "users", user)

	// Create a session
	session, sessionUser, token, err := manager.Create(ctx, "user1", nil)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session == nil {
		t.Fatal("Expected session, got nil")
	}

	if token == "" {
		t.Fatal("Expected token, got empty string")
	}

	if sessionUser == nil {
		t.Fatal("Expected user, got nil")
	}

	if sessionUser.ID != "user1" {
		t.Errorf("Expected user ID user1, got %s", sessionUser.ID)
	}

	// Retrieve the session
	retrievedSession, retrievedUser, err := manager.Get(ctx, token)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if retrievedSession == nil {
		t.Fatal("Expected session, got nil")
	}

	if retrievedSession.Token != token {
		t.Errorf("Expected token %s, got %s", token, retrievedSession.Token)
	}

	if retrievedUser == nil {
		t.Fatal("Expected user, got nil")
	}

	if retrievedUser.ID != "user1" {
		t.Errorf("Expected user ID user1, got %s", retrievedUser.ID)
	}
}

func TestManager_Delete(t *testing.T) {
	adapter := memory.New()
	defer adapter.Close()

	config := DefaultConfig()
	config.EnableRedisStore = false
	config.EnableCookieStore = false

	manager, err := NewManager(config, adapter)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Create a user first
	user := map[string]interface{}{
		"id":    "user1",
		"email": "test@example.com",
		"name":  "Test User",
	}
	adapter.Create(ctx, "users", user)

	// Create a session
	_, _, token, err := manager.Create(ctx, "user1", nil)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify session exists
	retrievedSession, _, err := manager.Get(ctx, token)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if retrievedSession == nil {
		t.Fatal("Expected session, got nil")
	}

	// Delete session
	err = manager.Delete(ctx, token)
	if err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	// Verify session is deleted
	retrievedSession, _, err = manager.Get(ctx, token)
	// Session not found is expected, not an error
	if err != nil && err != core.ErrSessionNotFound {
		t.Fatalf("Unexpected error getting session after delete: %v", err)
	}

	if retrievedSession != nil {
		t.Error("Expected nil session after delete, got session")
	}
}

func TestManager_DeleteByUserID(t *testing.T) {
	adapter := memory.New()
	defer adapter.Close()

	config := DefaultConfig()
	config.EnableRedisStore = false
	config.EnableCookieStore = false

	manager, err := NewManager(config, adapter)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Create a user first
	user := map[string]interface{}{
		"id":    "user1",
		"email": "test@example.com",
		"name":  "Test User",
	}
	adapter.Create(ctx, "users", user)

	// Create multiple sessions
	_, _, token1, err := manager.Create(ctx, "user1", nil)
	if err != nil {
		t.Fatalf("Failed to create session 1: %v", err)
	}

	_, _, token2, err := manager.Create(ctx, "user1", nil)
	if err != nil {
		t.Fatalf("Failed to create session 2: %v", err)
	}

	// Verify sessions exist
	session1, _, _ := manager.Get(ctx, token1)
	session2, _, _ := manager.Get(ctx, token2)

	if session1 == nil || session2 == nil {
		t.Fatal("Expected both sessions to exist")
	}

	// Delete all sessions for user
	err = manager.DeleteByUserID(ctx, "user1")
	if err != nil {
		t.Fatalf("Failed to delete sessions by user ID: %v", err)
	}

	// Verify sessions are deleted
	session1, _, _ = manager.Get(ctx, token1)
	session2, _, _ = manager.Get(ctx, token2)

	if session1 != nil || session2 != nil {
		t.Error("Expected all sessions to be deleted")
	}
}

func TestManager_SessionExpiration(t *testing.T) {
	adapter := memory.New()
	defer adapter.Close()

	config := DefaultConfig()
	config.EnableRedisStore = false
	config.EnableCookieStore = false
	config.ExpiresIn = 100 * time.Millisecond // Very short expiration

	manager, err := NewManager(config, adapter)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	// Create a user first
	user := map[string]interface{}{
		"id":    "user1",
		"email": "test@example.com",
		"name":  "Test User",
	}
	adapter.Create(ctx, "users", user)

	// Create a session
	_, _, token, err := manager.Create(ctx, "user1", nil)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Session should exist immediately
	session, _, err := manager.Get(ctx, token)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if session == nil {
		t.Fatal("Expected session, got nil")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Session should be expired
	session, _, err = manager.Get(ctx, token)
	// Session not found is expected for expired sessions
	if err != nil && err != core.ErrSessionNotFound {
		t.Fatalf("Unexpected error getting expired session: %v", err)
	}

	if session != nil {
		t.Error("Expected nil session after expiration, got session")
	}
}

func TestCookieStore_CreateAndGet(t *testing.T) {
	store := NewCookieStore("test-secret-key", "beaconauth")

	session := &core.Session{
		ID:        "session1",
		UserID:    "user1",
		Token:     "token123",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	user := &core.User{
		ID:    "user1",
		Email: "test@example.com",
		Name:  "Test User",
	}

	// Create token
	token, err := store.CreateToken(session, user)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	if token == "" {
		t.Fatal("Expected token, got empty string")
	}

	// Retrieve session
	ctx := context.Background()
	retrievedSession, retrievedUser, err := store.Get(ctx, token)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if retrievedSession == nil {
		t.Fatal("Expected session, got nil")
	}

	if retrievedSession.UserID != "user1" {
		t.Errorf("Expected user ID user1, got %s", retrievedSession.UserID)
	}

	if retrievedUser == nil {
		t.Fatal("Expected user, got nil")
	}

	if retrievedUser.Email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", retrievedUser.Email)
	}
}

func TestCookieStore_InvalidSignature(t *testing.T) {
	store := NewCookieStore("test-secret-key", "beaconauth")

	// Try to get a session with an invalid token
	ctx := context.Background()
	_, _, err := store.Get(ctx, "invalid.token")

	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}
}

func TestManager_StrategyDetermination(t *testing.T) {
	tests := []struct {
		name            string
		enableCookie    bool
		enableRedis     bool
		enableDB        bool
		redisAddr       string
		expectedStrategy Strategy
	}{
		{
			name:            "cookie only",
			enableCookie:    true,
			enableRedis:     false,
			enableDB:        false,
			expectedStrategy: StrategyCookieOnly,
		},
		{
			name:            "db only",
			enableCookie:    false,
			enableRedis:     false,
			enableDB:        true,
			expectedStrategy: StrategyDBOnly,
		},
		{
			name:            "redis first",
			enableCookie:    false,
			enableRedis:     true,
			enableDB:        true,
			redisAddr:       "localhost:6379",
			expectedStrategy: StrategyRedisFirst,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := memory.New()
			defer adapter.Close()

			config := DefaultConfig()
			config.EnableCookieStore = tt.enableCookie
			config.EnableRedisStore = tt.enableRedis
			config.EnableDBStore = tt.enableDB
			config.RedisAddr = tt.redisAddr

			manager, err := NewManager(config, adapter)
			if err != nil {
				// Skip if Redis connection fails (expected in test environments)
				if tt.enableRedis && tt.redisAddr != "" {
					t.Skipf("Skipping due to Redis connection failure: %v", err)
				}
				t.Fatalf("Failed to create manager: %v", err)
			}
			defer manager.Close()

			if manager.strategy != tt.expectedStrategy {
				t.Errorf("Expected strategy %v, got %v", tt.expectedStrategy, manager.strategy)
			}
		})
	}
}
