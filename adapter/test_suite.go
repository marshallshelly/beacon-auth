package adapter

import (
	"context"
	"testing"
	"time"

	"github.com/marshallshelly/beaconauth/core"
)

// TestSuite is a comprehensive test suite for adapter implementations
// Use this to ensure your adapter behaves correctly and consistently
type TestSuite struct {
	// Adapter to test
	Adapter core.Adapter

	// SetupFunc is called before each test to prepare the database
	// It should create any necessary tables/collections
	SetupFunc func(t *testing.T, adapter core.Adapter)

	// TeardownFunc is called after each test to clean up
	TeardownFunc func(t *testing.T, adapter core.Adapter)
}

// RunAll runs all tests in the suite
func (suite *TestSuite) RunAll(t *testing.T) {
	t.Run("Create", suite.TestCreate)
	t.Run("FindOne", suite.TestFindOne)
	t.Run("FindMany", suite.TestFindMany)
	t.Run("Update", suite.TestUpdate)
	t.Run("UpdateMany", suite.TestUpdateMany)
	t.Run("Delete", suite.TestDelete)
	t.Run("DeleteMany", suite.TestDeleteMany)
	t.Run("Count", suite.TestCount)
	t.Run("Operators", suite.TestOperators)
	t.Run("LimitOffset", suite.TestLimitOffset)
	t.Run("OrderBy", suite.TestOrderBy)
	t.Run("Transaction", suite.TestTransaction)
	t.Run("Ping", suite.TestPing)
}

// TestCreate tests the Create operation
func (suite *TestSuite) TestCreate(t *testing.T) {
	if suite.SetupFunc != nil {
		suite.SetupFunc(t, suite.Adapter)
	}
	if suite.TeardownFunc != nil {
		defer suite.TeardownFunc(t, suite.Adapter)
	}

	ctx := context.Background()

	data := map[string]interface{}{
		"id":             "user1",
		"email":          "test@example.com",
		"name":           "Test User",
		"email_verified": false,
		"created_at":     time.Now(),
		"updated_at":     time.Now(),
	}

	result, err := suite.Adapter.Create(ctx, "users", data)
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

// TestFindOne tests the FindOne operation
func (suite *TestSuite) TestFindOne(t *testing.T) {
	if suite.SetupFunc != nil {
		suite.SetupFunc(t, suite.Adapter)
	}
	if suite.TeardownFunc != nil {
		defer suite.TeardownFunc(t, suite.Adapter)
	}

	ctx := context.Background()

	// Create test data
	suite.Adapter.Create(ctx, "users", map[string]interface{}{
		"id":             "user1",
		"email":          "test@example.com",
		"name":           "Test User",
		"email_verified": false,
		"created_at":     time.Now(),
		"updated_at":     time.Now(),
	})

	// Test finding existing record
	query := &core.Query{
		Model: "users",
		Where: []core.WhereClause{
			{Field: "email", Operator: core.OpEqual, Value: "test@example.com"},
		},
	}

	result, err := suite.Adapter.FindOne(ctx, query)
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
	result, err = suite.Adapter.FindOne(ctx, query)
	if err != nil {
		t.Fatalf("FindOne failed: %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}

// TestFindMany tests the FindMany operation
func (suite *TestSuite) TestFindMany(t *testing.T) {
	if suite.SetupFunc != nil {
		suite.SetupFunc(t, suite.Adapter)
	}
	if suite.TeardownFunc != nil {
		defer suite.TeardownFunc(t, suite.Adapter)
	}

	ctx := context.Background()

	// Create test data
	suite.Adapter.Create(ctx, "users", map[string]interface{}{
		"id":     "user1",
		"email":  "user1@example.com",
		"active": true,
	})
	suite.Adapter.Create(ctx, "users", map[string]interface{}{
		"id":     "user2",
		"email":  "user2@example.com",
		"active": true,
	})
	suite.Adapter.Create(ctx, "users", map[string]interface{}{
		"id":     "user3",
		"email":  "user3@example.com",
		"active": false,
	})

	// Test finding multiple records
	query := &core.Query{
		Model: "users",
		Where: []core.WhereClause{
			{Field: "active", Operator: core.OpEqual, Value: true},
		},
	}

	results, err := suite.Adapter.FindMany(ctx, query)
	if err != nil {
		t.Fatalf("FindMany failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

// TestUpdate tests the Update operation
func (suite *TestSuite) TestUpdate(t *testing.T) {
	if suite.SetupFunc != nil {
		suite.SetupFunc(t, suite.Adapter)
	}
	if suite.TeardownFunc != nil {
		defer suite.TeardownFunc(t, suite.Adapter)
	}

	ctx := context.Background()

	// Create test data
	suite.Adapter.Create(ctx, "users", map[string]interface{}{
		"id":    "user1",
		"email": "test@example.com",
		"name":  "Old Name",
	})

	// Update
	query := &core.Query{
		Model: "users",
		Where: []core.WhereClause{
			{Field: "id", Operator: core.OpEqual, Value: "user1"},
		},
	}

	result, err := suite.Adapter.Update(ctx, query, map[string]interface{}{
		"name": "New Name",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if result["name"] != "New Name" {
		t.Errorf("Expected name=New Name, got %v", result["name"])
	}

	// Verify update persisted
	result, _ = suite.Adapter.FindOne(ctx, query)
	if result["name"] != "New Name" {
		t.Errorf("Update did not persist")
	}
}

// TestUpdateMany tests the UpdateMany operation
func (suite *TestSuite) TestUpdateMany(t *testing.T) {
	if suite.SetupFunc != nil {
		suite.SetupFunc(t, suite.Adapter)
	}
	if suite.TeardownFunc != nil {
		defer suite.TeardownFunc(t, suite.Adapter)
	}

	ctx := context.Background()

	// Create test data
	suite.Adapter.Create(ctx, "users", map[string]interface{}{
		"id":     "user1",
		"email":  "user1@example.com",
		"active": true,
	})
	suite.Adapter.Create(ctx, "users", map[string]interface{}{
		"id":     "user2",
		"email":  "user2@example.com",
		"active": true,
	})
	suite.Adapter.Create(ctx, "users", map[string]interface{}{
		"id":     "user3",
		"email":  "user3@example.com",
		"active": false,
	})

	query := &core.Query{
		Model: "users",
		Where: []core.WhereClause{
			{Field: "active", Operator: core.OpEqual, Value: true},
		},
	}

	count, err := suite.Adapter.UpdateMany(ctx, query, map[string]interface{}{
		"active": false,
	})
	if err != nil {
		t.Fatalf("UpdateMany failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 updates, got %d", count)
	}

	// Verify all active records were updated
	activeCount, _ := suite.Adapter.Count(ctx, &core.Query{
		Model: "users",
		Where: []core.WhereClause{
			{Field: "active", Operator: core.OpEqual, Value: true},
		},
	})

	if activeCount != 0 {
		t.Errorf("Expected 0 active users, got %d", activeCount)
	}
}

// TestDelete tests the Delete operation
func (suite *TestSuite) TestDelete(t *testing.T) {
	if suite.SetupFunc != nil {
		suite.SetupFunc(t, suite.Adapter)
	}
	if suite.TeardownFunc != nil {
		defer suite.TeardownFunc(t, suite.Adapter)
	}

	ctx := context.Background()

	// Create test data
	suite.Adapter.Create(ctx, "users", map[string]interface{}{
		"id":    "user1",
		"email": "test@example.com",
	})

	// Delete
	query := &core.Query{
		Model: "users",
		Where: []core.WhereClause{
			{Field: "id", Operator: core.OpEqual, Value: "user1"},
		},
	}

	err := suite.Adapter.Delete(ctx, query)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	result, _ := suite.Adapter.FindOne(ctx, query)
	if result != nil {
		t.Errorf("Record was not deleted")
	}
}

// TestDeleteMany tests the DeleteMany operation
func (suite *TestSuite) TestDeleteMany(t *testing.T) {
	if suite.SetupFunc != nil {
		suite.SetupFunc(t, suite.Adapter)
	}
	if suite.TeardownFunc != nil {
		defer suite.TeardownFunc(t, suite.Adapter)
	}

	ctx := context.Background()

	// Create test data
	suite.Adapter.Create(ctx, "users", map[string]interface{}{
		"id":     "user1",
		"email":  "user1@example.com",
		"active": true,
	})
	suite.Adapter.Create(ctx, "users", map[string]interface{}{
		"id":     "user2",
		"email":  "user2@example.com",
		"active": true,
	})
	suite.Adapter.Create(ctx, "users", map[string]interface{}{
		"id":     "user3",
		"email":  "user3@example.com",
		"active": false,
	})

	query := &core.Query{
		Model: "users",
		Where: []core.WhereClause{
			{Field: "active", Operator: core.OpEqual, Value: true},
		},
	}

	count, err := suite.Adapter.DeleteMany(ctx, query)
	if err != nil {
		t.Fatalf("DeleteMany failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 deletions, got %d", count)
	}

	// Verify count
	totalCount, _ := suite.Adapter.Count(ctx, &core.Query{Model: "users"})
	if totalCount != 1 {
		t.Errorf("Expected 1 remaining user, got %d", totalCount)
	}
}

// TestCount tests the Count operation
func (suite *TestSuite) TestCount(t *testing.T) {
	if suite.SetupFunc != nil {
		suite.SetupFunc(t, suite.Adapter)
	}
	if suite.TeardownFunc != nil {
		defer suite.TeardownFunc(t, suite.Adapter)
	}

	ctx := context.Background()

	// Create test data
	suite.Adapter.Create(ctx, "users", map[string]interface{}{
		"id":     "user1",
		"active": true,
	})
	suite.Adapter.Create(ctx, "users", map[string]interface{}{
		"id":     "user2",
		"active": true,
	})
	suite.Adapter.Create(ctx, "users", map[string]interface{}{
		"id":     "user3",
		"active": false,
	})

	query := &core.Query{
		Model: "users",
		Where: []core.WhereClause{
			{Field: "active", Operator: core.OpEqual, Value: true},
		},
	}

	count, err := suite.Adapter.Count(ctx, query)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected count=2, got %d", count)
	}
}

// TestOperators tests all query operators
func (suite *TestSuite) TestOperators(t *testing.T) {
	if suite.SetupFunc != nil {
		suite.SetupFunc(t, suite.Adapter)
	}
	if suite.TeardownFunc != nil {
		defer suite.TeardownFunc(t, suite.Adapter)
	}

	ctx := context.Background()

	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	// Create test data
	suite.Adapter.Create(ctx, "sessions", map[string]interface{}{
		"id":         "s1",
		"user_id":    "u1",
		"count":      10,
		"expires_at": future,
	})
	suite.Adapter.Create(ctx, "sessions", map[string]interface{}{
		"id":         "s2",
		"user_id":    "u2",
		"count":      5,
		"expires_at": past,
	})
	suite.Adapter.Create(ctx, "sessions", map[string]interface{}{
		"id":         "s3",
		"user_id":    "u3",
		"count":      15,
		"expires_at": future,
	})

	tests := []struct {
		name     string
		operator core.Operator
		field    string
		value    interface{}
		expected int
	}{
		{
			name:     "equal",
			operator: core.OpEqual,
			field:    "user_id",
			value:    "u1",
			expected: 1,
		},
		{
			name:     "not equal",
			operator: core.OpNotEqual,
			field:    "user_id",
			value:    "u1",
			expected: 2,
		},
		{
			name:     "greater than",
			operator: core.OpGreaterThan,
			field:    "count",
			value:    7,
			expected: 2,
		},
		{
			name:     "greater or equal",
			operator: core.OpGreaterOrEqual,
			field:    "count",
			value:    10,
			expected: 2,
		},
		{
			name:     "less than",
			operator: core.OpLessThan,
			field:    "count",
			value:    10,
			expected: 1,
		},
		{
			name:     "less or equal",
			operator: core.OpLessOrEqual,
			field:    "count",
			value:    10,
			expected: 2,
		},
		{
			name:     "IN",
			operator: core.OpIn,
			field:    "user_id",
			value:    []interface{}{"u1", "u2"},
			expected: 2,
		},
		{
			name:     "NOT IN",
			operator: core.OpNotIn,
			field:    "user_id",
			value:    []interface{}{"u1", "u2"},
			expected: 1,
		},
		{
			name:     "greater than (time)",
			operator: core.OpGreaterThan,
			field:    "expires_at",
			value:    now,
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

			results, err := suite.Adapter.FindMany(ctx, query)
			if err != nil {
				t.Fatalf("FindMany failed: %v", err)
			}

			if len(results) != tt.expected {
				t.Errorf("Expected %d results, got %d", tt.expected, len(results))
			}
		})
	}
}

// TestLimitOffset tests limit and offset functionality
func (suite *TestSuite) TestLimitOffset(t *testing.T) {
	if suite.SetupFunc != nil {
		suite.SetupFunc(t, suite.Adapter)
	}
	if suite.TeardownFunc != nil {
		defer suite.TeardownFunc(t, suite.Adapter)
	}

	ctx := context.Background()

	// Create 10 test records
	for i := 1; i <= 10; i++ {
		suite.Adapter.Create(ctx, "users", map[string]interface{}{
			"id":    i,
			"email": "user" + string(rune(i)) + "@example.com",
		})
	}

	// Test limit
	query := &core.Query{
		Model: "users",
		Limit: 5,
	}

	results, err := suite.Adapter.FindMany(ctx, query)
	if err != nil {
		t.Fatalf("FindMany failed: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results with limit, got %d", len(results))
	}

	// Test offset
	query.Offset = 3
	query.Limit = 0

	results, err = suite.Adapter.FindMany(ctx, query)
	if err != nil {
		t.Fatalf("FindMany failed: %v", err)
	}

	if len(results) != 7 {
		t.Errorf("Expected 7 results with offset=3, got %d", len(results))
	}

	// Test limit + offset
	query.Limit = 2
	query.Offset = 5

	results, err = suite.Adapter.FindMany(ctx, query)
	if err != nil {
		t.Fatalf("FindMany failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results with limit=2 offset=5, got %d", len(results))
	}
}

// TestOrderBy tests ordering functionality
func (suite *TestSuite) TestOrderBy(t *testing.T) {
	if suite.SetupFunc != nil {
		suite.SetupFunc(t, suite.Adapter)
	}
	if suite.TeardownFunc != nil {
		defer suite.TeardownFunc(t, suite.Adapter)
	}

	ctx := context.Background()

	// Create test data with different counts
	suite.Adapter.Create(ctx, "users", map[string]interface{}{
		"id":    "user1",
		"email": "c@example.com",
		"count": 3,
	})
	suite.Adapter.Create(ctx, "users", map[string]interface{}{
		"id":    "user2",
		"email": "a@example.com",
		"count": 1,
	})
	suite.Adapter.Create(ctx, "users", map[string]interface{}{
		"id":    "user3",
		"email": "b@example.com",
		"count": 2,
	})

	// Test ascending order
	query := &core.Query{
		Model: "users",
		OrderBy: []core.OrderBy{
			{Field: "count", Desc: false},
		},
	}

	results, err := suite.Adapter.FindMany(ctx, query)
	if err != nil {
		t.Fatalf("FindMany failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	// Verify order (should be 1, 2, 3)
	counts := []int{
		toInt(results[0]["count"]),
		toInt(results[1]["count"]),
		toInt(results[2]["count"]),
	}

	if counts[0] > counts[1] || counts[1] > counts[2] {
		t.Errorf("Results not in ascending order: %v", counts)
	}

	// Test descending order
	query.OrderBy[0].Desc = true

	results, err = suite.Adapter.FindMany(ctx, query)
	if err != nil {
		t.Fatalf("FindMany failed: %v", err)
	}

	counts = []int{
		toInt(results[0]["count"]),
		toInt(results[1]["count"]),
		toInt(results[2]["count"]),
	}

	if counts[0] < counts[1] || counts[1] < counts[2] {
		t.Errorf("Results not in descending order: %v", counts)
	}
}

// TestTransaction tests transaction functionality
func (suite *TestSuite) TestTransaction(t *testing.T) {
	if suite.SetupFunc != nil {
		suite.SetupFunc(t, suite.Adapter)
	}
	if suite.TeardownFunc != nil {
		defer suite.TeardownFunc(t, suite.Adapter)
	}

	ctx := context.Background()

	// Test successful transaction
	err := suite.Adapter.Transaction(ctx, func(tx core.Adapter) error {
		_, err := tx.Create(ctx, "users", map[string]interface{}{
			"id":    "user1",
			"email": "test1@example.com",
		})
		if err != nil {
			return err
		}

		_, err = tx.Create(ctx, "users", map[string]interface{}{
			"id":    "user2",
			"email": "test2@example.com",
		})
		return err
	})

	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	// Verify both records were created
	count, _ := suite.Adapter.Count(ctx, &core.Query{Model: "users"})
	if count != 2 {
		t.Errorf("Expected 2 records after transaction, got %d", count)
	}
}

// TestPing tests the Ping operation
func (suite *TestSuite) TestPing(t *testing.T) {
	ctx := context.Background()

	err := suite.Adapter.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

// Helper function to convert interface{} to int
func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}
