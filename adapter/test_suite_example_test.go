package adapter_test

import (
	"context"
	"testing"

	"github.com/marshallshelly/beacon-auth/adapter"
	"github.com/marshallshelly/beacon-auth/adapters/memory"
	"github.com/marshallshelly/beacon-auth/core"
)

// Example: How to use the TestSuite with your adapter
func TestMemoryAdapterWithTestSuite(t *testing.T) {
	memAdapter := memory.New()

	suite := &adapter.TestSuite{
		Adapter: memAdapter,
		SetupFunc: func(t *testing.T, a core.Adapter) {
			// For memory adapter, no setup needed as it's in-memory
			// For database adapters, you would create tables here
		},
		TeardownFunc: func(t *testing.T, a core.Adapter) {
			// Clean up by closing the adapter
			// For memory adapter, this clears all data
			a.Close()
		},
	}

	// Run all tests
	suite.RunAll(t)
}

// Example: Running individual tests
func TestMemoryAdapterIndividualTests(t *testing.T) {
	memAdapter := memory.New()

	suite := &adapter.TestSuite{
		Adapter: memAdapter,
	}

	// Run only specific tests
	t.Run("Create", suite.TestCreate)
	t.Run("FindOne", suite.TestFindOne)
	t.Run("Count", suite.TestCount)
}

// Example: Custom setup for database adapters
// This shows how you would use it with a real database
func ExampleTestSuite_withPostgres() {
	// This is example code, not runnable in tests

	/*
		import "github.com/marshallshelly/beacon-auth/adapters/postgres"

		func TestPostgresAdapter(t *testing.T) {
			ctx := context.Background()
			adapter, _ := postgres.New(ctx, &postgres.Config{
				Host:     "localhost",
				Database: "test_db",
				Username: "postgres",
				Password: "postgres",
			})

			suite := &adapter.TestSuite{
				Adapter: adapter,
				SetupFunc: func(t *testing.T, a core.Adapter) {
					// Create test tables
					// You can use raw SQL or migrations
				},
				TeardownFunc: func(t *testing.T, a core.Adapter) {
					// Drop test tables and close connection
					a.Close()
				},
			}

			suite.RunAll(t)
		}
	*/
}

// Example: Testing adapter-specific features
// The test suite covers common functionality, but adapters can have
// additional features that need specific tests
func TestMemoryAdapterSpecificFeatures(t *testing.T) {
	adapter := memory.New()
	ctx := context.Background()

	// Test memory-specific behavior
	// For example, testing that Close() actually clears data

	adapter.Create(ctx, "users", map[string]interface{}{
		"id":    "user1",
		"email": "test@example.com",
	})

	count, _ := adapter.Count(ctx, &core.Query{Model: "users"})
	if count != 1 {
		t.Errorf("Expected 1 user before close, got %d", count)
	}

	// Close should clear all data for memory adapter
	adapter.Close()

	// Create a new record after close
	_, err := adapter.Create(ctx, "users", map[string]interface{}{
		"id":    "user2",
		"email": "test2@example.com",
	})
	if err != nil {
		t.Fatalf("Create after close failed: %v", err)
	}

	// Should only have 1 record (the new one)
	count, _ = adapter.Count(ctx, &core.Query{Model: "users"})
	if count != 1 {
		t.Errorf("Expected 1 user after close and recreate, got %d", count)
	}
}
