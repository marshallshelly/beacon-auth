package memory

import (
	"context"
	"testing"
	"time"

	"github.com/marshallshelly/beaconauth/core"
)

func TestMemoryAdapter_Create(t *testing.T) {
	adapter := New()
	ctx := context.Background()

	data := map[string]interface{}{
		"id":    "user1",
		"email": "test@example.com",
		"name":  "Test User",
	}

	result, err := adapter.Create(ctx, "users", data)
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

func TestMemoryAdapter_FindOne(t *testing.T) {
	adapter := New()
	ctx := context.Background()

	// Create test data
	adapter.Create(ctx, "users", map[string]interface{}{
		"id":    "user1",
		"email": "test@example.com",
	})

	// Test finding existing record
	query := &core.Query{
		Model: "users",
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

func TestMemoryAdapter_FindMany(t *testing.T) {
	adapter := New()
	ctx := context.Background()

	// Create test data
	adapter.Create(ctx, "users", map[string]interface{}{"id": "user1", "active": true})
	adapter.Create(ctx, "users", map[string]interface{}{"id": "user2", "active": true})
	adapter.Create(ctx, "users", map[string]interface{}{"id": "user3", "active": false})

	// Test finding multiple records
	query := &core.Query{
		Model: "users",
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

func TestMemoryAdapter_Update(t *testing.T) {
	adapter := New()
	ctx := context.Background()

	// Create test data
	adapter.Create(ctx, "users", map[string]interface{}{
		"id":   "user1",
		"name": "Old Name",
	})

	// Update
	query := &core.Query{
		Model: "users",
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

func TestMemoryAdapter_Delete(t *testing.T) {
	adapter := New()
	ctx := context.Background()

	// Create test data
	adapter.Create(ctx, "users", map[string]interface{}{"id": "user1"})

	// Delete
	query := &core.Query{
		Model: "users",
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

func TestMemoryAdapter_Count(t *testing.T) {
	adapter := New()
	ctx := context.Background()

	// Create test data
	adapter.Create(ctx, "users", map[string]interface{}{"id": "user1", "active": true})
	adapter.Create(ctx, "users", map[string]interface{}{"id": "user2", "active": true})
	adapter.Create(ctx, "users", map[string]interface{}{"id": "user3", "active": false})

	query := &core.Query{
		Model: "users",
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

func TestMemoryAdapter_Operators(t *testing.T) {
	adapter := New()
	ctx := context.Background()

	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	// Create test data
	adapter.Create(ctx, "sessions", map[string]interface{}{
		"id":         "s1",
		"expires_at": future,
		"count":      10,
	})
	adapter.Create(ctx, "sessions", map[string]interface{}{
		"id":         "s2",
		"expires_at": past,
		"count":      5,
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
			name:     "greater than (number)",
			operator: core.OpGreaterThan,
			field:    "count",
			value:    7,
			expected: 1,
		},
		{
			name:     "less or equal (number)",
			operator: core.OpLessOrEqual,
			field:    "count",
			value:    10,
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := &core.Query{
				Model: "sessions",
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

func TestMemoryAdapter_LimitOffset(t *testing.T) {
	adapter := New()
	ctx := context.Background()

	// Create test data
	for i := 1; i <= 10; i++ {
		adapter.Create(ctx, "users", map[string]interface{}{"id": i})
	}

	// Test limit
	query := &core.Query{
		Model: "users",
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
