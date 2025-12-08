package mongodb

import (
	"context"
	"testing"
	"time"

	"github.com/marshallshelly/beacon-auth/core"
)

// getTestAdapter attempts to create a Mongo adapter; skips if unavailable.
func getTestAdapter(t *testing.T) *MongoAdapter {
	t.Helper()
	cfg := &Config{URI: "mongodb://localhost:27017", Database: "beaconauth_test"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ad, err := New(ctx, cfg)
	if err != nil {
		t.Skipf("skipping mongo tests: %v", err)
	}
	// Clean collections used in tests
	_ = ad.collection("users").Drop(ctx)
	_ = ad.collection("sessions").Drop(ctx)
	return ad
}

func TestMongoAdapter_CreateFindUpdateDelete(t *testing.T) {
	ad := getTestAdapter(t)
	ctx := context.Background()

	// Create
	u := map[string]interface{}{"id": "user1", "email": "test@example.com", "active": true}
	_, err := ad.Create(ctx, "users", u)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// FindOne
	res, err := ad.FindOne(ctx, QueryUsersEqual("email", "test@example.com"))
	if err != nil {
		t.Fatalf("findone failed: %v", err)
	}
	if res == nil || res["id"] != "user1" {
		t.Fatalf("expected user1, got %v", res)
	}

	// Update
	updated, err := ad.Update(ctx, QueryUsersEqual("id", "user1"), map[string]interface{}{"active": false})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated == nil || updated["active"] != false {
		t.Fatalf("expected active=false, got %v", updated)
	}

	// Count
	cnt, err := ad.Count(ctx, QueryUsersEqual("active", false))
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("expected count=1, got %d", cnt)
	}

	// Delete
	if err := ad.Delete(ctx, QueryUsersEqual("id", "user1")); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	res, err = ad.FindOne(ctx, QueryUsersEqual("id", "user1"))
	if err != nil {
		t.Fatalf("findone after delete failed: %v", err)
	}
	if res != nil {
		t.Fatalf("expected nil after delete, got %v", res)
	}
}

// helpers
func QueryUsersEqual(field string, value interface{}) *core.Query {
	return &core.Query{Model: "users", Where: []core.WhereClause{{Field: field, Operator: core.OpEqual, Value: value}}}
}
