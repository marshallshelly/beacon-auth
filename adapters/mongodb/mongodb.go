package mongodb

import (
	"context"
	"fmt"
	"strings"

	"github.com/marshallshelly/beacon-auth/core"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoAdapter implements the Adapter interface for MongoDB
type MongoAdapter struct {
	client   *mongo.Client
	database *mongo.Database
}

// Config holds MongoDB configuration
type Config struct {
	URI      string
	Database string
}

// New creates a new MongoDB adapter
func New(ctx context.Context, cfg *Config) (*MongoAdapter, error) {
	if cfg == nil || cfg.URI == "" || cfg.Database == "" {
		return nil, fmt.Errorf("mongodb config requires URI and Database")
	}
	clientOpts := options.Client().ApplyURI(cfg.URI)
	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}
	db := client.Database(cfg.Database)
	return &MongoAdapter{client: client, database: db}, nil
}

// ID returns the adapter identifier
func (m *MongoAdapter) ID() string { return "mongodb" }

func (m *MongoAdapter) collection(model string) *mongo.Collection {
	return m.database.Collection(model)
}

// Create inserts a document
func (m *MongoAdapter) Create(ctx context.Context, model string, data map[string]interface{}) (map[string]interface{}, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data provided")
	}
	// Insert as-is; driver codec will encode map[string]interface{}
	_, err := m.collection(model).InsertOne(ctx, data)
	if err != nil {
		return nil, err
	}
	// Return a copy of the provided data
	out := make(map[string]interface{}, len(data))
	for k, v := range data {
		out[k] = v
	}
	return out, nil
}

// FindOne finds a single document
func (m *MongoAdapter) FindOne(ctx context.Context, query *core.Query) (map[string]interface{}, error) {
	filter := buildFilter(query.Where)
	res := m.collection(query.Model).FindOne(ctx, filter)
	if res.Err() == mongo.ErrNoDocuments {
		return nil, nil
	}
	if res.Err() != nil {
		return nil, res.Err()
	}
	var doc bson.M
	if err := res.Decode(&doc); err != nil {
		return nil, err
	}
	return bsonMToMap(doc), nil
}

// FindMany finds documents matching the query
func (m *MongoAdapter) FindMany(ctx context.Context, query *core.Query) ([]map[string]interface{}, error) {
	filter := buildFilter(query.Where)
	opts := options.Find()
	// OrderBy
	if len(query.OrderBy) > 0 {
		sort := bson.D{}
		for _, ob := range query.OrderBy {
			if ob.Desc {
				sort = append(sort, bson.E{Key: ob.Field, Value: -1})
			} else {
				sort = append(sort, bson.E{Key: ob.Field, Value: 1})
			}
		}
		opts.SetSort(sort)
	}
	if query.Limit > 0 {
		opts.SetLimit(int64(query.Limit))
	}
	if query.Offset > 0 {
		opts.SetSkip(int64(query.Offset))
	}

	cursor, err := m.collection(query.Model).Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		results = append(results, bsonMToMap(doc))
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// Update updates a single document and returns it
func (m *MongoAdapter) Update(ctx context.Context, query *core.Query, data map[string]interface{}) (map[string]interface{}, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data provided")
	}
	filter := buildFilter(query.Where)
	update := bson.M{"$set": data}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	res := m.collection(query.Model).FindOneAndUpdate(ctx, filter, update, opts)
	if res.Err() == mongo.ErrNoDocuments {
		return nil, nil
	}
	if res.Err() != nil {
		return nil, res.Err()
	}
	var doc bson.M
	if err := res.Decode(&doc); err != nil {
		return nil, err
	}
	return bsonMToMap(doc), nil
}

// UpdateMany updates all matching documents
func (m *MongoAdapter) UpdateMany(ctx context.Context, query *core.Query, data map[string]interface{}) (int64, error) {
	if len(data) == 0 {
		return 0, fmt.Errorf("no data provided")
	}
	filter := buildFilter(query.Where)
	update := bson.M{"$set": data}
	res, err := m.collection(query.Model).UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, nil
}

// Delete deletes a single matching document
func (m *MongoAdapter) Delete(ctx context.Context, query *core.Query) error {
	filter := buildFilter(query.Where)
	_, err := m.collection(query.Model).DeleteOne(ctx, filter)
	return err
}

// DeleteMany deletes all matching documents
func (m *MongoAdapter) DeleteMany(ctx context.Context, query *core.Query) (int64, error) {
	filter := buildFilter(query.Where)
	res, err := m.collection(query.Model).DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
}

// Count returns the count of matching documents
func (m *MongoAdapter) Count(ctx context.Context, query *core.Query) (int64, error) {
	filter := buildFilter(query.Where)
	return m.collection(query.Model).CountDocuments(ctx, filter)
}

// Transaction executes a function in a transaction (best-effort)
func (m *MongoAdapter) Transaction(ctx context.Context, fn func(core.Adapter) error) error {
	// Use client-side transaction if available
	sess, err := m.client.StartSession()
	if err != nil {
		// Fallback: run without transaction
		return fn(m)
	}
	defer sess.EndSession(ctx)
	if err := sess.StartTransaction(); err != nil {
		return err
	}
	if err := fn(m); err != nil {
		_ = sess.AbortTransaction(ctx)
		return err
	}
	return sess.CommitTransaction(ctx)
}

// Ping checks connection
func (m *MongoAdapter) Ping(ctx context.Context) error { return m.client.Ping(ctx, nil) }

// Close disconnects client
func (m *MongoAdapter) Close() error { return m.client.Disconnect(context.Background()) }

// Helper: build filter from WhereClause
func buildFilter(where []core.WhereClause) bson.M {
	if len(where) == 0 {
		return bson.M{}
	}
	and := make([]bson.M, 0, len(where))
	for _, c := range where {
		and = append(and, clauseToFilter(c))
	}
	if len(and) == 1 {
		return and[0]
	}
	return bson.M{"$and": and}
}

func clauseToFilter(c core.WhereClause) bson.M {
	field := c.Field
	switch c.Operator {
	case core.OpEqual:
		return bson.M{field: bson.M{"$eq": c.Value}}
	case core.OpNotEqual:
		return bson.M{field: bson.M{"$ne": c.Value}}
	case core.OpGreaterThan:
		return bson.M{field: bson.M{"$gt": c.Value}}
	case core.OpGreaterOrEqual:
		return bson.M{field: bson.M{"$gte": c.Value}}
	case core.OpLessThan:
		return bson.M{field: bson.M{"$lt": c.Value}}
	case core.OpLessOrEqual:
		return bson.M{field: bson.M{"$lte": c.Value}}
	case core.OpLike:
		// Simple contains match (case-sensitive); use regex
		s, _ := c.Value.(string)
		pattern := s
		// Translate SQL LIKE %foo% to regex contains
		pattern = strings.ReplaceAll(pattern, "%", ".*")
		return bson.M{field: bson.M{"$regex": pattern}}
	case core.OpIn:
		return bson.M{field: bson.M{"$in": c.Value}}
	case core.OpNotIn:
		return bson.M{field: bson.M{"$nin": c.Value}}
	case core.OpIsNull:
		return bson.M{field: bson.M{"$eq": nil}}
	case core.OpIsNotNull:
		return bson.M{field: bson.M{"$ne": nil}}
	default:
		return bson.M{field: c.Value}
	}
}

func bsonMToMap(m bson.M) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	// Prefer "id" over "_id" if both exist; do nothing if only one exists
	// Keep _id as-is
	return out
}
