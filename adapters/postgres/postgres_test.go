package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/marshallshelly/beacon-auth/core"
)

// These tests require a running PostgreSQL instance
// Set the following environment variables to run:
//   PG_TEST_HOST (default: localhost)
//   PG_TEST_PORT (default: 5432)
//   PG_TEST_DB (default: beaconauth_test)
//   PG_TEST_USER (default: postgres)
//   PG_TEST_PASSWORD (default: postgres)
//
// Or skip tests with: go test -short

func getTestConfig() *Config {
	return &Config{
		Host:     getEnvOr("POSTGRES_HOST", "PG_TEST_HOST", "localhost"),
		Port:     getEnvIntOr("POSTGRES_PORT", "PG_TEST_PORT", 5432),
		Database: getEnvOr("POSTGRES_DB", "PG_TEST_DB", "beaconauth_test"),
		Username: getEnvOr("POSTGRES_USER", "PG_TEST_USER", "postgres"),
		Password: getEnvOr("POSTGRES_PASSWORD", "PG_TEST_PASSWORD", "postgres"),
		SSLMode:  "disable",
		MaxConns: 5,
		MinConns: 1,
	}
}

// getEnvOr checks multiple env var names, returning first non-empty value or default
func getEnvOr(primaryKey, secondaryKey, defaultValue string) string {
	if value := os.Getenv(primaryKey); value != "" {
		return value
	}
	if value := os.Getenv(secondaryKey); value != "" {
		return value
	}
	return defaultValue
}

// getEnvIntOr checks multiple env var names for int values
func getEnvIntOr(primaryKey, secondaryKey string, defaultValue int) int {
	if value := os.Getenv(primaryKey); value != "" {
		var i int
		if _, err := fmt.Sscanf(value, "%d", &i); err == nil {
			return i
		}
	}
	if value := os.Getenv(secondaryKey); value != "" {
		var i int
		if _, err := fmt.Sscanf(value, "%d", &i); err == nil {
			return i
		}
	}
	return defaultValue
}

func setupTestDB(t *testing.T) *PostgresAdapter {
	if testing.Short() {
		t.Skip("Skipping PostgreSQL tests in short mode")
	}

	ctx := context.Background()
	cfg := getTestConfig()

	adapter, err := New(ctx, cfg)
	if err != nil {
		t.Skipf("Could not connect to PostgreSQL: %v. Set PG_TEST_* env vars or run with -short to skip", err)
	}

	// Create test tables
	createTablesSQL := `
		DROP TABLE IF EXISTS test_users CASCADE;
		DROP TABLE IF EXISTS test_sessions CASCADE;
		DROP TABLE IF EXISTS test_items CASCADE;

		CREATE TABLE test_users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			name TEXT,
			email_verified BOOLEAN DEFAULT false,
			created_at TIMESTAMP,
			updated_at TIMESTAMP
		);

		CREATE TABLE test_sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			token TEXT UNIQUE NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			ip_address TEXT,
			user_agent TEXT,
			created_at TIMESTAMP,
			updated_at TIMESTAMP
		);

		CREATE TABLE test_items (
			id TEXT PRIMARY KEY,
			name TEXT,
			active BOOLEAN DEFAULT true,
			count INTEGER DEFAULT 0,
			created_at TIMESTAMP
		);
	`

	_, err = adapter.pool.Exec(ctx, createTablesSQL)
	if err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	return adapter
}

func cleanupTestDB(t *testing.T, adapter *PostgresAdapter) {
	ctx := context.Background()

	_, err := adapter.pool.Exec(ctx, `
		DROP TABLE IF EXISTS test_users CASCADE;
		DROP TABLE IF EXISTS test_sessions CASCADE;
		DROP TABLE IF EXISTS test_items CASCADE;
	`)
	if err != nil {
		t.Errorf("Failed to cleanup test tables: %v", err)
	}

	adapter.Close()
}

func TestPostgresAdapter_Create(t *testing.T) {
	adapter := setupTestDB(t)
	defer cleanupTestDB(t, adapter)

	ctx := context.Background()

	data := map[string]interface{}{
		"id":             "user1",
		"email":          "test@example.com",
		"name":           "Test User",
		"email_verified": false,
		"created_at":     time.Now(),
		"updated_at":     time.Now(),
	}

	result, err := adapter.Create(ctx, "test_users", data)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if result["id"] != "user1" {
		t.Errorf("Expected id=user1, got %v", result["id"])
	}
	if result["email"] != "test@example.com" {
		t.Errorf("Expected email=test@example.com, got %v", result["email"])
	}
}

func TestPostgresAdapter_FindOne(t *testing.T) {
	adapter := setupTestDB(t)
	defer cleanupTestDB(t, adapter)

	ctx := context.Background()

	// Create test data
	adapter.Create(ctx, "test_users", map[string]interface{}{
		"id":             "user1",
		"email":          "test@example.com",
		"name":           "Test User",
		"email_verified": false,
		"created_at":     time.Now(),
		"updated_at":     time.Now(),
	})

	// Test finding existing record
	query := &core.Query{
		Model: "test_users",
		Where: []core.WhereClause{
			{Field: "email", Operator: core.OpEqual, Value: "test@example.com"},
		},
	}

	result, err := adapter.FindOne(ctx, query)
	if err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result["id"] != "user1" {
		t.Errorf("Expected id=user1, got %v", result["id"])
	}

	// Test finding non-existent record
	query.Where[0].Value = "nonexistent@example.com"
	result, err = adapter.FindOne(ctx, query)
	if err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}

func TestPostgresAdapter_FindMany(t *testing.T) {
	adapter := setupTestDB(t)
	defer cleanupTestDB(t, adapter)

	ctx := context.Background()

	// Create test data
	adapter.Create(ctx, "test_items", map[string]interface{}{
		"id":         "item1",
		"name":       "Item 1",
		"active":     true,
		"count":      10,
		"created_at": time.Now(),
	})
	adapter.Create(ctx, "test_items", map[string]interface{}{
		"id":         "item2",
		"name":       "Item 2",
		"active":     true,
		"count":      20,
		"created_at": time.Now(),
	})
	adapter.Create(ctx, "test_items", map[string]interface{}{
		"id":         "item3",
		"name":       "Item 3",
		"active":     false,
		"count":      5,
		"created_at": time.Now(),
	})

	// Test finding multiple records
	query := &core.Query{
		Model: "test_items",
		Where: []core.WhereClause{
			{Field: "active", Operator: core.OpEqual, Value: true},
		},
	}

	results, err := adapter.FindMany(ctx, query)
	if err != nil {
		t.Fatalf("FindMany failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestPostgresAdapter_Update(t *testing.T) {
	adapter := setupTestDB(t)
	defer cleanupTestDB(t, adapter)

	ctx := context.Background()

	// Create test data
	adapter.Create(ctx, "test_users", map[string]interface{}{
		"id":             "user1",
		"email":          "test@example.com",
		"name":           "Old Name",
		"email_verified": false,
		"created_at":     time.Now(),
		"updated_at":     time.Now(),
	})

	// Update
	query := &core.Query{
		Model: "test_users",
		Where: []core.WhereClause{
			{Field: "id", Operator: core.OpEqual, Value: "user1"},
		},
	}

	result, err := adapter.Update(ctx, query, map[string]interface{}{
		"name": "New Name",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if result["name"] != "New Name" {
		t.Errorf("Expected name=New Name, got %v", result["name"])
	}

	// Verify update persisted
	result, _ = adapter.FindOne(ctx, query)
	if result["name"] != "New Name" {
		t.Errorf("Update did not persist")
	}
}

func TestPostgresAdapter_Delete(t *testing.T) {
	adapter := setupTestDB(t)
	defer cleanupTestDB(t, adapter)

	ctx := context.Background()

	// Create test data
	adapter.Create(ctx, "test_users", map[string]interface{}{
		"id":             "user1",
		"email":          "test@example.com",
		"name":           "Test User",
		"email_verified": false,
		"created_at":     time.Now(),
		"updated_at":     time.Now(),
	})

	// Delete
	query := &core.Query{
		Model: "test_users",
		Where: []core.WhereClause{
			{Field: "id", Operator: core.OpEqual, Value: "user1"},
		},
	}

	err := adapter.Delete(ctx, query)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	result, _ := adapter.FindOne(ctx, query)
	if result != nil {
		t.Errorf("Record was not deleted")
	}
}

func TestPostgresAdapter_Count(t *testing.T) {
	adapter := setupTestDB(t)
	defer cleanupTestDB(t, adapter)

	ctx := context.Background()

	// Create test data
	adapter.Create(ctx, "test_items", map[string]interface{}{
		"id":         "item1",
		"name":       "Item 1",
		"active":     true,
		"count":      10,
		"created_at": time.Now(),
	})
	adapter.Create(ctx, "test_items", map[string]interface{}{
		"id":         "item2",
		"name":       "Item 2",
		"active":     true,
		"count":      20,
		"created_at": time.Now(),
	})
	adapter.Create(ctx, "test_items", map[string]interface{}{
		"id":         "item3",
		"name":       "Item 3",
		"active":     false,
		"count":      5,
		"created_at": time.Now(),
	})

	query := &core.Query{
		Model: "test_items",
		Where: []core.WhereClause{
			{Field: "active", Operator: core.OpEqual, Value: true},
		},
	}

	count, err := adapter.Count(ctx, query)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected count=2, got %d", count)
	}
}

func TestPostgresAdapter_Operators(t *testing.T) {
	adapter := setupTestDB(t)
	defer cleanupTestDB(t, adapter)

	ctx := context.Background()

	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	// Create test data
	adapter.Create(ctx, "test_sessions", map[string]interface{}{
		"id":         "s1",
		"user_id":    "u1",
		"token":      "token1",
		"expires_at": future,
		"created_at": now,
		"updated_at": now,
	})
	adapter.Create(ctx, "test_sessions", map[string]interface{}{
		"id":         "s2",
		"user_id":    "u2",
		"token":      "token2",
		"expires_at": past,
		"created_at": now,
		"updated_at": now,
	})

	tests := []struct {
		name     string
		operator core.Operator
		field    string
		value    interface{}
		expected int
	}{
		{
			name:     "greater than (time)",
			operator: core.OpGreaterThan,
			field:    "expires_at",
			value:    now,
			expected: 1,
		},
		{
			name:     "less than (time)",
			operator: core.OpLessThan,
			field:    "expires_at",
			value:    now,
			expected: 1,
		},
		{
			name:     "IN operator",
			operator: core.OpIn,
			field:    "user_id",
			value:    []interface{}{"u1", "u2"},
			expected: 2,
		},
		{
			name:     "NOT IN operator",
			operator: core.OpNotIn,
			field:    "user_id",
			value:    []interface{}{"u3", "u4"},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &core.Query{
				Model: "test_sessions",
				Where: []core.WhereClause{
					{Field: tt.field, Operator: tt.operator, Value: tt.value},
				},
			}

			results, err := adapter.FindMany(ctx, query)
			if err != nil {
				t.Fatalf("FindMany failed: %v", err)
			}

			if len(results) != tt.expected {
				t.Errorf("Expected %d results, got %d", tt.expected, len(results))
			}
		})
	}
}

func TestPostgresAdapter_LimitOffset(t *testing.T) {
	adapter := setupTestDB(t)
	defer cleanupTestDB(t, adapter)

	ctx := context.Background()

	// Create test data
	for i := 1; i <= 10; i++ {
		adapter.Create(ctx, "test_items", map[string]interface{}{
			"id":         fmt.Sprintf("item%d", i),
			"name":       fmt.Sprintf("Item %d", i),
			"active":     true,
			"count":      i,
			"created_at": time.Now(),
		})
	}

	// Test limit
	query := &core.Query{
		Model: "test_items",
		Limit: 5,
	}

	results, err := adapter.FindMany(ctx, query)
	if err != nil {
		t.Fatalf("FindMany failed: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results with limit, got %d", len(results))
	}

	// Test offset
	query.Offset = 3
	query.Limit = 0

	results, err = adapter.FindMany(ctx, query)
	if err != nil {
		t.Fatalf("FindMany failed: %v", err)
	}

	if len(results) != 7 {
		t.Errorf("Expected 7 results with offset=3, got %d", len(results))
	}

	// Test limit + offset
	query.Limit = 2
	query.Offset = 5

	results, err = adapter.FindMany(ctx, query)
	if err != nil {
		t.Fatalf("FindMany failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results with limit=2 offset=5, got %d", len(results))
	}
}

func TestPostgresAdapter_Transaction(t *testing.T) {
	adapter := setupTestDB(t)
	defer cleanupTestDB(t, adapter)

	ctx := context.Background()

	// Test successful transaction
	err := adapter.Transaction(ctx, func(tx core.Adapter) error {
		_, err := tx.Create(ctx, "test_users", map[string]interface{}{
			"id":             "user1",
			"email":          "test1@example.com",
			"name":           "User 1",
			"email_verified": false,
			"created_at":     time.Now(),
			"updated_at":     time.Now(),
		})
		if err != nil {
			return err
		}

		_, err = tx.Create(ctx, "test_users", map[string]interface{}{
			"id":             "user2",
			"email":          "test2@example.com",
			"name":           "User 2",
			"email_verified": false,
			"created_at":     time.Now(),
			"updated_at":     time.Now(),
		})
		return err
	})

	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	// Verify both records were created
	count, _ := adapter.Count(ctx, &core.Query{Model: "test_users"})
	if count != 2 {
		t.Errorf("Expected 2 records after transaction, got %d", count)
	}

	// Test rollback on error
	err = adapter.Transaction(ctx, func(tx core.Adapter) error {
		_, err := tx.Create(ctx, "test_users", map[string]interface{}{
			"id":             "user3",
			"email":          "test3@example.com",
			"name":           "User 3",
			"email_verified": false,
			"created_at":     time.Now(),
			"updated_at":     time.Now(),
		})
		if err != nil {
			return err
		}

		// Force an error (duplicate email)
		_, err = tx.Create(ctx, "test_users", map[string]interface{}{
			"id":             "user4",
			"email":          "test1@example.com", // Duplicate
			"name":           "User 4",
			"email_verified": false,
			"created_at":     time.Now(),
			"updated_at":     time.Now(),
		})
		return err
	})

	if err == nil {
		t.Fatal("Expected transaction to fail")
	}

	// Verify rollback - should still be 2 records
	count, _ = adapter.Count(ctx, &core.Query{Model: "test_users"})
	if count != 2 {
		t.Errorf("Expected 2 records after rollback, got %d", count)
	}
}

func TestPostgresAdapter_Ping(t *testing.T) {
	adapter := setupTestDB(t)
	defer cleanupTestDB(t, adapter)

	ctx := context.Background()

	err := adapter.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}
